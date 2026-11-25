package e2e

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// levanta el binario de la API contra PostgreSQL, recorre un escenario con
// concurrencia y rechazos esperados, y compara un reporte determinista

var actualizarGolden = flag.Bool("actualizar", false, "reescribe testdata/escenario.golden.md con el resultado actual")

const (
	goldenEscenario = "testdata/escenario.golden.md"
	imagenPostgres  = "postgres:17.11@sha256:ff87138949ad9a46f32ff23bd6852b52ce024a13dffe9b4087be98292cdd3b4f"
	kidE2E          = "e2e"
)

func TestE2EBinarioPostgres(t *testing.T) {
	dbURL := baseDeDatosE2E(t)
	db := migrarBase(t, dbURL)
	jwksURL, firmar := servidorJWKS(t)
	base := levantarBinario(t, dbURL, jwksURL)

	operador := firmar("operador-ocr", []string{"lotes:operar", "lotes:leer"})
	lector := firmar("lector-reportes", []string{"registros:leer"})
	c := &clienteE2E{t: t, base: base}
	var r reporteE2E

	r.titulo("Estado del servicio")
	r.linea("GET /healthz → %d", c.pedir("GET", "/healthz", "", "", nil).estado)
	r.linea("GET /readyz → %d", c.pedir("GET", "/readyz", "", "", nil).estado)

	r.titulo("Autorización")
	sinToken := c.pedir("GET", "/api/v1/lotes", "", "", nil)
	r.linea("Sin token → %d (%s)", sinToken.estado, sinToken.tipo)
	sinPermiso := c.pedir("POST", "/api/v1/lotes", lector, `{"origen":"x","referencia_externa":"x","registros":[{"folio":"a","nombre":"b","correo":"c@d.cl"}]}`, nil)
	r.linea("Token sin lotes:operar → %d (%s)", sinPermiso.estado, sinPermiso.tipo)

	r.titulo("Lote con registros válidos e inválidos")
	lote := `{"origen":"extractor-ocr","referencia_externa":"e2e-001","registros":[` +
		`{"folio":" doc-001 ","nombre":"  Camila   Soto ","correo":" CAMILA@EXAMPLE.COM "},` +
		`{"folio":"doc-002","nombre":"Diego Pérez","correo":"diego@example.com"},` +
		`{"folio":"doc-003","nombre":" ","correo":"prueba@example.com"},` +
		`{"folio":"","nombre":"Sin Folio","correo":"no-es-correo"},` +
		`{"folio":" DOC-001","nombre":"Camila Soto","correo":"camila@example.com"}]}`
	creado := c.pedir("POST", "/api/v1/lotes", operador, lote, map[string]string{"Idempotency-Key": "e2e-001"})
	r.linea("POST /api/v1/lotes → %d", creado.estado)
	r.linea("Recibidos: %v, aceptados: %v, rechazados: %v", creado.json["recibidos"], creado.json["aceptados"], creado.json["rechazados"])
	for _, rech := range lista(creado.json["rechazos"]) {
		r.linea("- Rechazo posición %v (folio %q): %s", rech["posicion"], rech["folio"], strings.Join(textos(rech["motivos"]), ", "))
	}
	idLote, _ := creado.json["id"].(string)
	registros := c.pedir("GET", "/api/v1/registros?lote_id="+idLote, lector, "", nil)
	r.linea("Registros guardados:")
	for _, reg := range lista(registros.json["elementos"]) {
		r.linea("- %v | %v | %v", reg["folio"], reg["nombre"], reg["correo"])
	}

	r.titulo("Reintentos")
	repetido := c.pedir("POST", "/api/v1/lotes", operador, lote, map[string]string{"Idempotency-Key": "e2e-001"})
	r.linea("Misma clave y contenido → %d, mismo lote: %v", repetido.estado, repetido.json["id"] == idLote)
	conflicto := c.pedir("POST", "/api/v1/lotes", operador, `{"origen":"otro","referencia_externa":"x","registros":[{"folio":"x-1","nombre":"X","correo":"x@x.cl"}]}`, map[string]string{"Idempotency-Key": "e2e-001"})
	r.linea("Misma clave con otro contenido → %d (%s)", conflicto.estado, conflicto.tipo)

	r.titulo("Cinco lotes simultáneos con el mismo folio")
	const simultaneos = 5
	var wg sync.WaitGroup
	resultados := make([]respuestaE2E, simultaneos)
	for i := range simultaneos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cuerpo := fmt.Sprintf(`{"origen":"emisor-%d","referencia_externa":"conc-%d","registros":[`+
				`{"folio":"DOC-900","nombre":"Folio Compartido","correo":"compartido@example.com"},`+
				`{"folio":"DOC-90%d","nombre":"Folio Propio","correo":"propio@example.com"}]}`, i, i, i+1)
			resultados[i] = c.pedir("POST", "/api/v1/lotes", operador, cuerpo, nil)
		}()
	}
	wg.Wait()
	var lineas, motivos []string
	for _, res := range resultados {
		lineas = append(lineas, fmt.Sprintf("POST → %d, aceptados: %v, rechazados: %v", res.estado, res.json["aceptados"], res.json["rechazados"]))
		for _, rech := range lista(res.json["rechazos"]) {
			motivos = append(motivos, textos(rech["motivos"])...)
		}
	}
	slices.Sort(lineas)
	for _, l := range lineas {
		r.linea("%s", l)
	}
	compartido := c.pedir("GET", "/api/v1/registros?folio=doc-900", lector, "", nil)
	r.linea("Registros con folio DOC-900: %v", compartido.json["total"])
	r.linea("Motivo de los rechazos: %s", strings.Join(slices.Compact(slices.Sorted(slices.Values(motivos))), ", "))

	r.titulo("Auditoría")
	filas, err := db.Query(`SELECT accion, actor, count(*) FROM auditoria GROUP BY accion, actor ORDER BY accion, actor`)
	if err != nil {
		t.Fatalf("consultando auditoría: %v", err)
	}
	for filas.Next() {
		var accion, actor string
		var n int
		if err := filas.Scan(&accion, &actor, &n); err != nil {
			t.Fatal(err)
		}
		r.linea("- %s por %s: %d", accion, actor, n)
	}
	_ = filas.Close()
	var sinRequestID int
	if err := db.QueryRow(`SELECT count(*) FROM auditoria WHERE request_id = ''`).Scan(&sinRequestID); err != nil {
		t.Fatal(err)
	}
	r.linea("Eventos sin request_id: %d", sinRequestID)

	r.titulo("Errores")
	for _, caso := range []struct{ nombre, metodo, ruta, cuerpo, ct, token string }{
		{"Lote sin registros", "POST", "/api/v1/lotes", `{"origen":"x","referencia_externa":"x","registros":[]}`, "application/json", operador},
		{"CSV sin columna correo", "POST", "/api/v1/lotes", "folio,nombre\na,b\n", "text/csv", operador},
		{"Content-Type no soportado", "POST", "/api/v1/lotes", "hola", "text/plain", operador},
		{"Lote inexistente", "GET", "/api/v1/lotes/00000000-0000-4000-8000-000000000000", "", "", operador},
		{"Id que no es UUID", "GET", "/api/v1/registros/no-es-uuid", "", "", lector},
	} {
		res := c.pedirTipo(caso.metodo, caso.ruta, caso.token, caso.cuerpo, caso.ct, nil)
		r.linea("%s → %d (%s)", caso.nombre, res.estado, res.tipo)
	}

	r.comparar(t)
}

func baseDeDatosE2E(t *testing.T) string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	sinDocker := func(motivo any) {
		if os.Getenv("CI") != "" {
			t.Fatalf("el e2e necesita Docker para Testcontainers: %v", motivo)
		}
		t.Skipf("Docker no disponible, se omite el e2e (define TEST_DATABASE_URL o abre Docker): %v", motivo)
	}
	defer func() {
		if r := recover(); r != nil {
			sinDocker(r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, imagenPostgres,
		tcpostgres.WithDatabase("gestor_e2e"),
		tcpostgres.WithUsername("gestor"),
		tcpostgres.WithPassword("gestor-e2e"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		sinDocker(err)
	}
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })
	url, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	return url
}

func migrarBase(t *testing.T, url string) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, filepath.Join("..", "db", "migrations")); err != nil {
		t.Fatalf("aplicando migraciones: %v", err)
	}
	return db
}

func servidorJWKS(t *testing.T) (string, func(sub string, permisos []string) string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwks, _ := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "OKP", "crv": "Ed25519", "alg": "EdDSA", "use": "sig", "kid": kidE2E,
		"x": base64.RawURLEncoding.EncodeToString(pub),
	}}})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwks)
	}))
	t.Cleanup(srv.Close)

	firmar := func(sub string, permisos []string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{
			"sub": sub, "permisos": permisos,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		})
		tok.Header["kid"] = kidE2E
		firmado, err := tok.SignedString(priv)
		if err != nil {
			t.Fatal(err)
		}
		return firmado
	}
	return srv.URL, firmar
}

func levantarBinario(t *testing.T, dbURL, jwksURL string) string {
	t.Helper()
	binario := os.Getenv("E2E_API_BIN")
	if binario == "" {
		binario = filepath.Join(t.TempDir(), "api")
		args := []string{"build", "-o", binario}
		if os.Getenv("GOCOVERDIR") != "" {
			args = append(args, "-cover")
		}
		compilar := exec.Command("go", append(args, "./cmd/api")...)
		compilar.Dir = ".."
		if salida, err := compilar.CombinedOutput(); err != nil {
			t.Fatalf("compilando la API: %v\n%s", err, salida)
		}
	}

	puerto := puertoLibre(t)
	var logs bytes.Buffer
	cmd := exec.Command(binario)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(),
		"API_PORT="+puerto,
		"API_ENV=test",
		"DATABASE_URL="+dbURL,
		"AUTH_JWKS_URL="+jwksURL,
		"FRONTEND_URL=http://localhost:5120",
		"TZ=America/Santiago",
		"LOG_LEVEL=info",
	)
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(syscall.SIGINT)
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("logs de la API:\n%s", logs.String())
		}
	})

	base := "http://127.0.0.1:" + puerto
	limite := time.Now().Add(30 * time.Second)
	for time.Now().Before(limite) {
		if res, err := http.Get(base + "/readyz"); err == nil {
			_ = res.Body.Close()
			if res.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("la API no quedó lista en 30 s\n%s", logs.String())
	return ""
}

func puertoLibre(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

type clienteE2E struct {
	t    *testing.T
	base string
}

type respuestaE2E struct {
	estado int
	tipo   string
	json   map[string]any
}

func (c *clienteE2E) pedir(metodo, ruta, token, cuerpo string, encabezados map[string]string) respuestaE2E {
	ct := ""
	if cuerpo != "" {
		ct = "application/json"
	}
	return c.pedirTipo(metodo, ruta, token, cuerpo, ct, encabezados)
}

func (c *clienteE2E) pedirTipo(metodo, ruta, token, cuerpo, contentType string, encabezados map[string]string) respuestaE2E {
	c.t.Helper()
	var lector io.Reader
	if cuerpo != "" {
		lector = strings.NewReader(cuerpo)
	}
	req, err := http.NewRequest(metodo, c.base+ruta, lector)
	if err != nil {
		c.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "session", Value: token})
	}
	for k, v := range encabezados {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", metodo, ruta, err)
	}
	defer res.Body.Close()
	out := respuestaE2E{estado: res.StatusCode, tipo: strings.Split(res.Header.Get("Content-Type"), ";")[0]}
	_ = json.NewDecoder(res.Body).Decode(&out.json)
	return out
}

func lista(v any) []map[string]any {
	crudos, _ := v.([]any)
	out := make([]map[string]any, 0, len(crudos))
	for _, c := range crudos {
		if m, ok := c.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func textos(v any) []string {
	crudos, _ := v.([]any)
	out := make([]string, 0, len(crudos))
	for _, c := range crudos {
		out = append(out, fmt.Sprint(c))
	}
	return out
}

type reporteE2E struct {
	b strings.Builder
}

func (r *reporteE2E) titulo(texto string) {
	if r.b.Len() > 0 {
		r.b.WriteString("\n")
	}
	fmt.Fprintf(&r.b, "## %s\n\n", texto)
}

func (r *reporteE2E) linea(formato string, args ...any) {
	fmt.Fprintf(&r.b, formato+"\n", args...)
}

func (r *reporteE2E) comparar(t *testing.T) {
	t.Helper()
	obtenido := "# Flujo de carga y consulta de lotes\n\n" + r.b.String()
	if *actualizarGolden {
		if err := os.WriteFile(goldenEscenario, []byte(obtenido), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	esperado, err := os.ReadFile(goldenEscenario)
	if err != nil {
		t.Fatalf("no existe %s; genéralo con -actualizar: %v", goldenEscenario, err)
	}
	if string(esperado) != obtenido {
		t.Errorf("el reporte no coincide con %s\n--- esperado\n%s\n--- obtenido\n%s", goldenEscenario, esperado, obtenido)
	}
}
