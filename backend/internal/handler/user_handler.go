package handler

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"github.com/saku-730/bio-occurrence/backend/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.svc.Register(req)
	if err != nil {
		// 重複エラーかどうか判定してステータスコードを変えるとより親切
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "ユーザー登録成功なのだ！",
		"user":    user, // パスワードハッシュはJSONタグで除外されているので安全
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.svc.Login(req)
	if err != nil {
		// 認証失敗は 401 Unauthorized
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "トークン生成失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success login",
		"user":    user,
		"token":   token, 
	})
}
