package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_CookieAuth(t *testing.T) {
	t.Parallel()
	t.Setenv("JWT_SECRET", "cookiekey")

	tok, err := MakeToken(42, "user", time.Hour)
	if err != nil {
		t.Fatalf("MakeToken: %v", err)
	}

	called := false
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := ContextUserID(r.Context()); !ok || id != 42 {
			t.Fatalf("user id not in context: %v %v", id, ok)
		}
		called = true
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "Authorization", Value: tok})
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status: %d", rr.Code)
	}
	if !called {
		t.Fatalf("handler not called")
	}
}

func TestMiddleware_Unauthorized(t *testing.T) {
	t.Parallel()
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
