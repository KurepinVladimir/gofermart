package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrNoSecret = errors.New("JWT_SECRET is not set")

func MakeToken(userID int64, login string, ttl time.Duration) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", ErrNoSecret
	}
	claims := jwt.MapClaims{
		"sub":   userID,
		"login": login,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseToken возвращает claims по строке токена
func ParseToken(token string) (jwt.MapClaims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, ErrNoSecret
	}
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// защитимся от другого алгоритма подписи
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !t.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("no claims")
	}
	return claims, nil
}
