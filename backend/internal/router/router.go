package router

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
	"github.com/saku-730/bio-occurrence/backend/internal/middleware"
	"time"

	"github.com/gin-contrib/cors"
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

	api := r.Group("/api")

	{
		api.GET("/occurrences", occHandler.GetAll)
		api.GET("/occurrences/:id", occHandler.GetDetail)
		api.GET("/search", occHandler.Search)

	//	authorized := api.Group("/")
	//	authorized.Use(middleware.AuthRequired())
//		{
//			authorized.POST("/occurrences", occHandler.Create)
//			authorized.PUT("/occurrences/:id", occHandler.Update)
//			authorized.DELETE("/occurrences/:id", occHandler.Delete)
//		}


	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
	}
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())

		{
			protected.POST("/occurrences", occHandler.Create)
			protected.PUT("/occurrences/:id", occHandler.Update)
			protected.DELETE("/occurrences/:id", occHandler.Delete)
		}
	}

	return r
}
