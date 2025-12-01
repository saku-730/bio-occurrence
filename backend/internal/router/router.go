package router

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
//	"github.com/saku-730/bio-occurrence/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRouter(occHandler *handler.OccurrenceHandler) *gin.Engine {
	r := gin.Default()
	
	// Middleware適用
//	r.Use(middleware.Cors()) 

	// ルーティング定義
	api := r.Group("/api")
	{
		api.POST("/occurrences", occHandler.Create)
		api.GET("/occurrences", occHandler.GetAll)
		api.GET("/occurrences/:id", occHandler.GetDetail)
		api.PUT("/occurrences/:id", occHandler.Update)
		api.DELETE("/occurrences/:id", occHandler.Delete)
	}

	return r
}
