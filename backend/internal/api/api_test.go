package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/netip"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"golang.org/x/crypto/bcrypt"

	"workoutapp/internal/auth"
	"workoutapp/internal/media"
	"workoutapp/internal/store"
)

const testPassword = "correct horse"

type env struct {
	srv    *httptest.Server
	st     *store.Store
	dbPath string
}

func newEnv(t *testing.T, proxies ...netip.Prefix) *env {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ms, err := media.New(filepath.Join(dir, "rec"))
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	a, err := auth.New(st, auth.Options{User: "jakob", PasswordHash: string(hash), TrustedProxies: proxies})
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"index.html":                  {Data: []byte("<html>app</html>")},
		"_app/immutable/entry/app.js": {Data: []byte("console.log(1)")},
	}
	h := New(Options{Store: st, Media: ms, Auth: a, MaxUploadBytes: 1 << 20, Static: static})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &env{srv: srv, st: st, dbPath: dbPath}
}

func (e *env) client() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func (e *env) do(t *testing.T, c *http.Client, method, path string, body any) (*http.Response, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func (e *env) login(t *testing.T) *http.Client {
	t.Helper()
	c := e.client()
	resp, b := e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "jakob", "password": testPassword})
	if resp.StatusCode != 200 {
		t.Fatalf("login: %d %s", resp.StatusCode, b)
	}
	return c
}

func TestUnauthenticatedIsRejected(t *testing.T) {
	e := newEnv(t)
	c := e.client()
	for _, p := range []string{"/api/auth/me", "/api/exercises", "/api/workouts", "/api/sessions", "/api/recordings", "/api/recordings/1/file", "/api/settings"} {
		resp, _ := e.do(t, c, "GET", p, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: got %d, want 401", p, resp.StatusCode)
		}
	}
	// Health und Frontend sind öffentlich.
	if resp, _ := e.do(t, c, "GET", "/api/health", nil); resp.StatusCode != 200 {
		t.Errorf("health: %d", resp.StatusCode)
	}
	if resp, b := e.do(t, c, "GET", "/workouts/3", nil); resp.StatusCode != 200 || !strings.Contains(string(b), "app") {
		t.Errorf("spa fallback: %d %s", resp.StatusCode, b)
	}
}

func TestLoginFlow(t *testing.T) {
	e := newEnv(t)
	c := e.client()
	resp, _ := e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "jakob", "password": "wrong"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong password: %d", resp.StatusCode)
	}
	resp, _ = e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "other", "password": testPassword})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong user: %d", resp.StatusCode)
	}
	resp, _ = e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "Jakob", "password": testPassword})
	if resp.StatusCode != 200 {
		t.Fatalf("login: %d", resp.StatusCode)
	}
	var cookie *http.Cookie
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.CookieName {
			cookie = ck
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie flags: %+v", cookie)
	}
	if resp, _ := e.do(t, c, "GET", "/api/auth/me", nil); resp.StatusCode != 200 {
		t.Fatalf("me after login: %d", resp.StatusCode)
	}

	// Gestohlenes Cookie in einem zweiten Client funktioniert bis zum Logout …
	c2 := e.client()
	c2.Jar.SetCookies(mustURL(e.srv.URL), []*http.Cookie{{Name: auth.CookieName, Value: cookie.Value}})
	if resp, _ := e.do(t, c2, "GET", "/api/auth/me", nil); resp.StatusCode != 200 {
		t.Fatalf("second client: %d", resp.StatusCode)
	}
	if resp, _ := e.do(t, c, "POST", "/api/auth/logout", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d", resp.StatusCode)
	}
	// … danach ist das Token serverseitig ungültig.
	if resp, _ := e.do(t, c2, "GET", "/api/auth/me", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token still valid after logout: %d", resp.StatusCode)
	}
}

func TestLoginRateLimit(t *testing.T) {
	e := newEnv(t)
	c := e.client()
	for i := 0; i < 5; i++ {
		resp, _ := e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "jakob", "password": "nope"})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d", i, resp.StatusCode)
		}
	}
	resp, _ := e.do(t, c, "POST", "/api/auth/login", map[string]string{"username": "jakob", "password": testPassword})
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 even with correct password, got %d", resp.StatusCode)
	}
	// Gefälschter X-Forwarded-For darf das Limit nicht umgehen (keine TRUSTED_PROXIES gesetzt).
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/login", strings.NewReader(`{"username":"jakob","password":"correct horse"}`))
	req.Header.Set("X-Forwarded-For", "10.9.8.7")
	r2, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("XFF bypassed rate limit: %d", r2.StatusCode)
	}
}

func TestTrustedProxyHeaders(t *testing.T) {
	// httptest-Clients kommen von 127.0.0.1 → als Caddy-Proxy eintragen.
	e := newEnv(t, netip.MustParsePrefix("127.0.0.1/32"))
	c := e.client()
	login := func(clientIP, password string) *http.Response {
		req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/login",
			strings.NewReader(`{"username":"jakob","password":"`+password+`"}`))
		req.Header.Set("X-Forwarded-For", clientIP)
		req.Header.Set("X-Forwarded-Proto", "https")
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp
	}
	// Rate-Limit gilt pro echter Client-IP (aus X-Forwarded-For), nicht pro Proxy-IP.
	for i := 0; i < 5; i++ {
		login("192.168.178.50", "nope")
	}
	if r := login("192.168.178.50", testPassword); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("limited client: %d", r.StatusCode)
	}
	r := login("192.168.178.51", testPassword)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("other client: %d", r.StatusCode)
	}
	var secure bool
	for _, ck := range r.Cookies() {
		if ck.Name == auth.CookieName {
			secure = ck.Secure
		}
	}
	if !secure {
		t.Fatal("cookie should be Secure behind HTTPS proxy")
	}
}

func TestUntrustedProxyHeadersIgnored(t *testing.T) {
	e := newEnv(t, netip.MustParsePrefix("192.168.178.20/32"))
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/auth/login",
		strings.NewReader(`{"username":"jakob","password":"correct horse"}`))
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := e.client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.CookieName && ck.Secure {
			t.Fatal("X-Forwarded-Proto from untrusted client must be ignored")
		}
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	e := newEnv(t)
	c := e.login(t)
	// Alle Sessions künstlich ablaufen lassen.
	if err := expireAll(e.dbPath); err != nil {
		t.Fatal(err)
	}
	if resp, _ := e.do(t, c, "GET", "/api/auth/me", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expired session accepted: %d", resp.StatusCode)
	}
}

func TestWorkoutCRUD(t *testing.T) {
	e := newEnv(t)
	c := e.login(t)
	resp, b := e.do(t, c, "POST", "/api/workouts", map[string]any{"name": "Neu", "structure": map[string]any{"version": 1, "blocks": []any{}}})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", resp.StatusCode, b)
	}
	var wo store.Workout
	json.Unmarshal(b, &wo)

	resp, _ = e.do(t, c, "POST", "/api/workouts", map[string]any{"name": "Kaputt", "structure": []any{}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid structure accepted: %d", resp.StatusCode)
	}
	resp, b = e.do(t, c, "POST", "/api/workouts/"+itoa(wo.ID)+"/duplicate", nil)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(string(b), "(Kopie)") {
		t.Fatalf("duplicate: %d %s", resp.StatusCode, b)
	}
	if resp, _ := e.do(t, c, "DELETE", "/api/workouts/"+itoa(wo.ID), nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if resp, _ := e.do(t, c, "GET", "/api/workouts/"+itoa(wo.ID), nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get deleted: %d", resp.StatusCode)
	}
}

func TestRecordingUploadAndRange(t *testing.T) {
	e := newEnv(t)
	c := e.login(t)
	video := bytes.Repeat([]byte("0123456789"), 100) // 1000 Bytes

	upload := func(mime string, data []byte) (*http.Response, []byte) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		mw.WriteField("exerciseId", "1")
		mw.WriteField("durationSec", "12.5")
		mw.WriteField("label", "Swings Satz 1")
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="file"; filename="clip"`)
		h.Set("Content-Type", mime)
		pw, _ := mw.CreatePart(h)
		pw.Write(data)
		mw.Close()
		req, _ := http.NewRequest("POST", e.srv.URL+"/api/recordings", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, b
	}

	if resp, _ := upload("text/plain", video); resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("non-video accepted: %d", resp.StatusCode)
	}
	if resp, _ := upload("video/webm", make([]byte, 2<<20)); resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized accepted: %d", resp.StatusCode)
	}

	resp, b := upload("video/webm;codecs=vp9", video)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload: %d %s", resp.StatusCode, b)
	}
	var rec store.Recording
	json.Unmarshal(b, &rec)
	if rec.SizeBytes != 1000 || rec.Mime != "video/webm" || rec.DurationSec != 12.5 || rec.ExerciseName == nil {
		t.Fatalf("unexpected recording %+v", rec)
	}

	req, _ := http.NewRequest("GET", e.srv.URL+"/api/recordings/"+itoa(rec.ID)+"/file", nil)
	req.Header.Set("Range", "bytes=10-19")
	r2, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(r2.Body)
	r2.Body.Close()
	if r2.StatusCode != http.StatusPartialContent || string(body) != "0123456789" {
		t.Fatalf("range: %d %q", r2.StatusCode, body)
	}
	if ct := r2.Header.Get("Content-Type"); ct != "video/webm" {
		t.Fatalf("content type %q", ct)
	}

	if resp, _ := e.do(t, c, "DELETE", "/api/recordings/"+itoa(rec.ID), nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if resp, _ := e.do(t, c, "GET", "/api/recordings/"+itoa(rec.ID)+"/file", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("file after delete: %d", resp.StatusCode)
	}
}

func TestStaticCaching(t *testing.T) {
	e := newEnv(t)
	resp, _ := e.do(t, e.client(), "GET", "/_app/immutable/entry/app.js", nil)
	if !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("immutable cache header missing: %q", resp.Header.Get("Cache-Control"))
	}
	if resp, _ := e.do(t, e.client(), "GET", "/_app/immutable/missing.js", nil); resp.StatusCode != 404 {
		t.Fatalf("missing asset: %d", resp.StatusCode)
	}
}

// expireAll lässt alle Login-Sessions über eine zweite DB-Verbindung ablaufen.
func expireAll(dbPath string) error {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(context.Background(), `UPDATE auth_sessions SET expires_at = '2000-01-01T00:00:00Z'`)
	return err
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
