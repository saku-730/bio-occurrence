package main

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"github.com/saku-730/bio-occurrence/backend/internal/router"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"fmt"
)

// 設定定数 (本来は環境変数から読むべき)
const (
	MeiliURL    = "http://localhost:7700"
	MeiliKey    = "masterKey123"
	FusekiURL  = "http://localhost:3030/biodb"
	FusekiUser = "admin"
	FusekiPass = "admin123"
)

func main() {
	// 1. 依存関係の組み立て (DI)
	// Repository -> Service -> Handler -> Router
	repo := repository.NewOccurrenceRepository(FusekiURL, FusekiUser, FusekiPass)
	searchRepo := repository.NewSearchRepository(MeiliURL, MeiliKey)

	svc := service.NewOccurrenceService(repo, searchRepo)

	h := handler.NewOccurrenceHandler(svc)
	
	r := router.SetupRouter(h)

	// 2. サーバー起動
	fmt.Println("🚀 APIサーバー起動: http://localhost:8080")
	r.Run(":8080")
}
