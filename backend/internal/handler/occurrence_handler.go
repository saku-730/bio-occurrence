package handler

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"net/http"

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

	id, err := h.svc.Register(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "登録成功", "id": id})
}

// GET /api/occurrences
func (h *OccurrenceHandler) GetAll(c *gin.Context) {
	list, err := h.svc.GetAll()
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

	if err := h.svc.Modify(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// DELETE /api/occurrences/:id
func (h *OccurrenceHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Remove(id); err != nil {
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

func (h *OccurrenceHandler) Search(c *gin.Context) {
	query := c.Query("q") // URLの ?q=... を取得

	// Service経由で検索実行
	docs, err := h.svc.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 結果を返す (空の場合は [] が返る)
	c.JSON(http.StatusOK, docs)
}

