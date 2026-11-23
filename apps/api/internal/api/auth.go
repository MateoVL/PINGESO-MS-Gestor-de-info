package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// Permisos declarados por recurso (AGENTS.md sección 6: Controles de aplicación).
const (
	PermLotesOperar          = "lotes:operar"
	PermLotesLeer            = "lotes:leer"
	PermLotesAdministrar     = "lotes:administrar"
	PermRegistrosLeer        = "registros:leer"
	PermRegistrosAdministrar = "registros:administrar"
)

// Nombres de cookie HttpOnly aceptados para la sesión JWT.
var sessionCookieNames = []string{"session", "access_token"}

type principalContextKey struct{}

// Principal representa al usuario o servicio autenticado por JWT.
type Principal struct {
	Subject  string
	Rol      string
	Permisos []string
}

// HasPermission verifica si el principal posee el permiso requerido o el permiso
// administrativo del mismo recurso (<recurso>:administrar).
func (p Principal) HasPermission(required string) bool {
	if required == "" {
		return true
	}
	adminPerm := ""
	if recurso, _, ok := strings.Cut(required, ":"); ok && recurso != "" {
		adminPerm = recurso + ":administrar"
	}
	for _, perm := range p.Permisos {
		if perm == required || (adminPerm != "" && perm == adminPerm) {
			return true
		}
	}
	return false
}

// WithPrincipal guarda el principal autenticado en el contexto.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// PrincipalFromContext extrae el principal autenticado del contexto.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(Principal)
	return p, ok
}

type authClaims struct {
	jwt.RegisteredClaims
	Rol         string   `json:"rol,omitempty"`
	Role        string   `json:"role,omitempty"`
	Permisos    []string `json:"permisos,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	Scope       string   `json:"scope,omitempty"`
}

// Authenticator valida JWT firmados con EdDSA o RS256 usando JWKS o un jwt.Keyfunc.
type Authenticator struct {
	keyfunc jwt.Keyfunc
	jwksURL string
}

// NewJWKSAuthenticator construye un Authenticator respaldado por AUTH_JWKS_URL con caché automático.
// Si el servicio de identidad aún no está disponible al arrancar, reintenta al autenticar peticiones.
func NewJWKSAuthenticator(ctx context.Context, jwksURL string) (*Authenticator, error) {
	jwksURL = strings.TrimSpace(jwksURL)
	if jwksURL == "" {
		return nil, errors.New("AUTH_JWKS_URL es obligatoria para inicializar el validador JWT")
	}
	kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return &Authenticator{jwksURL: jwksURL}, fmt.Errorf("inicializando JWKS desde %s: %w", jwksURL, err)
	}
	return &Authenticator{keyfunc: kf.Keyfunc, jwksURL: jwksURL}, nil
}

// NewKeyfuncAuthenticator construye un Authenticator con un jwt.Keyfunc explícito (útil en pruebas).
func NewKeyfuncAuthenticator(kf jwt.Keyfunc) *Authenticator {
	return &Authenticator{keyfunc: kf}
}

// Authenticate extrae y valida el JWT desde la cookie HttpOnly de sesión o desde
// el encabezado Authorization: Bearer (reenvío entre microservicios).
func (a *Authenticator) Authenticate(r *http.Request) (Principal, error) {
	if a == nil {
		return Principal{}, ErrUnauthorized
	}
	kf := a.keyfunc
	if kf == nil && a.jwksURL != "" {
		resolved, err := keyfunc.NewDefaultCtx(r.Context(), []string{a.jwksURL})
		if err != nil {
			return Principal{}, ErrUnauthorized
		}
		kf = resolved.Keyfunc
		a.keyfunc = kf
	}
	if kf == nil {
		return Principal{}, ErrUnauthorized
	}
	rawToken := extractToken(r)
	if rawToken == "" {
		return Principal{}, ErrUnauthorized
	}

	var claims authClaims
	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		kf,
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg(), jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return Principal{}, ErrUnauthorized
	}

	rol := claims.Rol
	if rol == "" {
		rol = claims.Role
	}

	perms := collectPermissions(claims)
	return Principal{
		Subject:  claims.Subject,
		Rol:      rol,
		Permisos: perms,
	}, nil
}

func extractToken(r *http.Request) string {
	for _, name := range sessionCookieNames {
		if c, err := r.Cookie(name); err == nil && strings.TrimSpace(c.Value) != "" {
			return strings.TrimSpace(c.Value)
		}
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}

func collectPermissions(c authClaims) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			out = append(out, p)
		}
	}
	for _, p := range c.Permisos {
		add(p)
	}
	for _, p := range c.Permissions {
		add(p)
	}
	for _, p := range strings.Fields(c.Scope) {
		add(p)
	}
	return out
}
