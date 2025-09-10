package auth

import (
	"context"
	"net/http"
	"strings"
)

// ключ для userID в контексте
type ctxKey string

const userIDKey ctxKey = "userID"

func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func ContextUserID(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(int64)
	return id, ok
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")
		claims, err := ParseToken(token) // из internal/auth/jwt.go
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// из MakeToken мы кладём sub как число (float64 в mapclaims)
		sub, ok := claims["sub"].(float64)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := WithUserID(r.Context(), int64(sub))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
