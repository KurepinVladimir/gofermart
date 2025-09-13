package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrNoSecret = errors.New("JWT_SECRET is not set") // оставим для совместимости (не будет срабатывать)
	secretOnce  sync.Once
	secretKey   []byte
)

func getSecret() ([]byte, error) {
	var err error
	secretOnce.Do(func() {
		s := os.Getenv("JWT_SECRET")
		if s == "" {
			// генерим 32 байта и кодируем — достаточно для HS256
			buf := make([]byte, 32)
			_, _ = rand.Read(buf)
			s = base64.StdEncoding.EncodeToString(buf)
		}
		secretKey = []byte(s)
	})
	if len(secretKey) == 0 {
		err = ErrNoSecret
	}
	return secretKey, err
}

func MakeToken(userID int64, login string, ttl time.Duration) (string, error) {
	secret, err := getSecret()
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub":   userID,
		"login": login,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(secret)
}

func ParseToken(token string) (jwt.MapClaims, error) {
	secret, err := getSecret()
	if err != nil {
		return nil, err
	}
	t, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
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
