package service

import (
	"context"
	"net/http"
	"time"
)

// DefaultHTTPTimeout define el timeout por defecto (5 s) según sección 2 de AGENTS.md
const DefaultHTTPTimeout = 5 * time.Second

type ctxKey string

const (
	RequestIDContextKey ctxKey = "request_id"
	AuthTokenContextKey ctxKey = "auth_token"
)

// NewHTTPClient construye un cliente HTTP con timeout explícito de 5 s y propagación de JWT y X-Request-ID
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   DefaultHTTPTimeout,
		Transport: &propagatingTransport{base: http.DefaultTransport},
	}
}

type propagatingTransport struct {
	base http.RoundTripper
}

func (t *propagatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if reqID := RequestIDFromContext(req.Context()); reqID != "" && clone.Header.Get("X-Request-ID") == "" {
		clone.Header.Set("X-Request-ID", reqID)
	}
	if token := AuthTokenFromContext(req.Context()); token != "" && clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", token)
	}
	return t.base.RoundTrip(clone)
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDContextKey, id)
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return v
	}
	return ""
}

func WithAuthToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, AuthTokenContextKey, token)
}

func AuthTokenFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(AuthTokenContextKey).(string); ok {
		return v
	}
	return ""
}

