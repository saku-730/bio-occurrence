package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// 秘密鍵（本番では環境変数から読むべきだけど、一旦定数で）
var jwtSecret = []byte("super_secret_key_CHANGE_THIS")

// Claims: トークンの中身（ユーザーIDなど）
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// トークン生成
func GenerateToken(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24時間有効
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// トークン検証（パース）
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
