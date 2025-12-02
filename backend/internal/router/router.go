package router

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
	"time"

	"github.com/gin-contrib/cors" // ★これを使っているか確認！
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	occHandler *handler.OccurrenceHandler,
	authHandler *handler.AuthHandler,
) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		// AllowOrigins ではなく AllowAllOrigins を使う
		AllowAllOrigins:  true,
		
		// メソッドも全部許可
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		
		// ヘッダーも主要なものは全部許可
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		
		// ブラウザに「OKだよ」と見せるヘッダー
		ExposeHeaders:    []string{"Content-Length"},
		
		// クッキーなどを許可するか（AllOrigins:true の時は false にしないと怒られることがあるので false 推奨）
		AllowCredentials: false, 
		
		MaxAge:           12 * time.Hour,
	}))

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}
	// APIルート定義
	api := r.Group("/api")
	{
		api.POST("/occurrences", occHandler.Create)
		api.GET("/occurrences", occHandler.GetAll)
		api.GET("/occurrences/:id", occHandler.GetDetail)
		api.PUT("/occurrences/:id", occHandler.Update)
		api.DELETE("/occurrences/:id", occHandler.Delete)

		api.GET("/search", occHandler.Search)
	}

	return r
}
