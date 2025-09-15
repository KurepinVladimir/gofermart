package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KurepinVladimir/gofermart/internal/logger"
)

func init() {
	_ = logger.Init() // чтобы логгер не был nil в путях с логированием
}

type jsonMap = map[string]any

func TestHandlers_Register_And_Login(t *testing.T) {
	t.Parallel()
	t.Setenv("JWT_SECRET", "hsecret")

	repo := newMemRepo()
	svc := NewService(repo)
	h := NewHandlers(svc)

	// REGISTER (200 + token + Set-Cookie + Authorization header)
	body := []byte(`{"login":"ann","password":"p"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)

	if rr.Code != 200 {
		t.Fatalf("register status: %d, body=%s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type: %s", ct)
	}
	if c := rr.Header().Get("Set-Cookie"); c == "" {
		t.Fatalf("cookie not set")
	}
	if a := rr.Header().Get("Authorization"); a == "" {
		t.Fatalf("Authorization header not set")
	}
	var resp jsonMap
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := resp["token"].(string); !ok {
		t.Fatalf("no token field")
	}

	// LOGIN ok (200 + token)
	body = []byte(`{"login":"ann","password":"p"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != 200 {
		t.Fatalf("login status: %d", rr.Code)
	}

	// LOGIN bad password (401)
	body = []byte(`{"login":"ann","password":"wrong"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	h.Login(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("login wrong status: %d", rr.Code)
	}
}

func TestHandlers_Register_Duplicate(t *testing.T) {
	t.Parallel()
	t.Setenv("JWT_SECRET", "hsecret2")

	repo := newMemRepo()
	svc := NewService(repo)
	h := NewHandlers(svc)

	body := []byte(`{"login":"dup","password":"p"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.Register(rr, req)
	if rr.Code != 200 {
		t.Fatalf("first register status: %d", rr.Code)
	}

	// повтор
	req2 := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.Register(rr2, req2)
	if rr2.Code != http.StatusConflict {
		t.Fatalf("second register status: %d", rr2.Code)
	}

	// sanity: выданный ранее токен можно распарсить
	var resp jsonMap
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	tok := resp["token"].(string)
	if _, err := ParseToken(tok); err != nil {
		t.Fatalf("ParseToken after register: %v", err)
	}
	_ = tok
	_ = time.Second
}
