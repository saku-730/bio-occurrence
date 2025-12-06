package handler

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"github.com/saku-730/bio-occurrence/backend/internal/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type OccurrenceHandler struct {
	svc service.OccurrenceService
}

func NewOccurrenceHandler(svc service.OccurrenceService) *OccurrenceHandler {
	return &OccurrenceHandler{svc: svc}
}

// POST /api/occurrences
func (h *OccurrenceHandler) Create(c *gin.Context) {
	var req model.OccurrenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id, err := h.svc.Register(userID.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "登録成功", "id": id})
}

// GET /api/occurrences
func (h *OccurrenceHandler) GetAll(c *gin.Context) {
	// ★修正: 任意認証でユーザーIDを取得して渡す
	userID := h.getOptionalUserID(c)
	
	list, err := h.svc.GetAll(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// GET /api/occurrences/:id
func (h *OccurrenceHandler) GetDetail(c *gin.Context) {
	id := c.Param("id")
	detail, err := h.svc.GetDetail(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// PUT /api/occurrences/:id
func (h *OccurrenceHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.OccurrenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.svc.Modify(userID.(string), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DELETE /api/occurrences/:id
func (h *OccurrenceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.svc.Remove(userID.(string), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "削除成功"})
}

// GET /api/taxons/:id
func (h *OccurrenceHandler) GetTaxonStats(c *gin.Context) {
	id := c.Param("id")
	stats, err := h.svc.GetTaxonStats(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GET /api/search
func (h *OccurrenceHandler) Search(c *gin.Context) {
	query := c.Query("q")
	taxonQuery := c.Query("taxon")
	
	userID := h.getOptionalUserID(c)

	// Service経由で検索実行 (userIDも渡す)
	docs, err := h.svc.Search(query, taxonQuery, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// ---------------------------------------------------
// Helper Methods
// ---------------------------------------------------

// getOptionalUserID: トークンがあればユーザーIDを返し、なければ空文字を返す
func (h *OccurrenceHandler) getOptionalUserID(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	// トークンを検証してIDを取り出す
	claims, err := utils.ParseToken(parts[1])
	if err != nil {
		return "" // 無効なトークンなら未ログイン扱い
	}

	return claims.UserID
}
