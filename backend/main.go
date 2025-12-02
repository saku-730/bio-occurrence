package main

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"github.com/saku-730/bio-occurrence/backend/internal/router"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"github.com/saku-730/bio-occurrence/backend/internal/infrastructure"
	"fmt"
)

// 設定定数 (本来は環境変数から読むべき)
const (
	MeiliURL   = "http://localhost:7700"
	MeiliKey   = "masterKey123"
	FusekiURL  = "http://localhost:3030/biodb"
	FusekiUser = "admin"
	FusekiPass = "admin123"
	PGHost = "localhost"
	PGPort = "5432"
	PGUser = "bio_user"
	PGPass = "14afqrzv" // docker-compose.ymlと合わせる！
	PGDB   = "bio_auth"
)

func main() {
	pgDB := infrastructure.NewPostgresDB(PGHost, PGPort, PGUser, PGPass, PGDB)

	// 1. 依存関係の組み立て (DI)
	// Repository -> Service -> Handler -> Router
	userRepo := repository.NewUserRepository(pgDB)
	repo := repository.NewOccurrenceRepository(FusekiURL, FusekiUser, FusekiPass)
	searchRepo := repository.NewSearchRepository(MeiliURL, MeiliKey)

	authSvc := service.NewAuthService(userRepo)
	svc := service.NewOccurrenceService(repo, searchRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	h := handler.NewOccurrenceHandler(svc)
	
	r := router.SetupRouter(h,authHandler)

	// 2. サーバー起動
	fmt.Println("🚀 APIサーバー起動: http://localhost:8080")
	r.Run(":8080")
}
