package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zyy125/im-system/internal/apperr"
	"github.com/zyy125/im-system/pkg/utils"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID string, secret string, expiry time.Duration) (string, string, error) {
	now := time.Now()
	jti := utils.GenerateUUID()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "im-system",
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return tokenString, jti, nil
}

func ParseJWT(tokenString string, secret string) (*Claims, error) {
	claims := &Claims{}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, apperr.TokenInvalid()
	}
	return claims, nil
}
