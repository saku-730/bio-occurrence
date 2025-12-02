package middleware

import (
	"github.com/saku-730/bio-occurrence/backend/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 認証ミドルウェア
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. ヘッダーから Authorization を取得
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証トークンが必要なのだ"})
			c.Abort()
			return
		}

		// 2. "Bearer <token>" の形式かチェック
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "トークンの形式が不正なのだ"})
			c.Abort()
			return
		}

		// 3. トークンを検証
		tokenString := parts[1]
		claims, err := utils.ParseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンなのだ"})
			c.Abort()
			return
		}

		// 4. 成功！ユーザーIDをコンテキストに保存（後でハンドラーで使うため）
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
