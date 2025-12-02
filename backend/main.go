package main

import (
	"github.com/saku-730/bio-occurrence/backend/internal/handler"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"github.com/saku-730/bio-occurrence/backend/internal/router"
	"github.com/saku-730/bio-occurrence/backend/internal/service"
	"github.com/saku-730/bio-occurrence/backend/internal/infrastructure"
	"fmt"
	"log"
	"os"
)

// 設定定数 (本来は環境変数から読むべき)
const (
	PGHost = "localhost"
	PGPort = "5432"
	PGUser = "bio_user"
	PGPass = "14afqrzv" // docker-compose.ymlと合わせる！
	PGDB   = "bio_auth"
)

func main() {
	meiliURL := getEnv("NEXT_PUBLIC_MEILI_URL")
	meiliKey := getEnv("NEXT_PUBLIC_MEILI_KEY")
	fusekiURL := getEnv("FUSEKI_URL")
	fusekiUser := getEnv("FUSEKI_USER")
	fusekiPass := getEnv("FUSEKI_PASSWORD")

	pgDB := infrastructure.NewPostgresDB(PGHost, PGPort, PGUser, PGPass, PGDB)
	// 1. 依存関係の組み立て (DI)
	userRepo := repository.NewUserRepository(pgDB)
	repo := repository.NewOccurrenceRepository(fusekiURL, fusekiUser, fusekiPass)
	searchRepo := repository.NewSearchRepository(meiliURL, meiliKey)

	authSvc := service.NewAuthService(userRepo)
	svc := service.NewOccurrenceService(repo, searchRepo)

	authHandler := handler.NewAuthHandler(authSvc)
	h := handler.NewOccurrenceHandler(svc)
	
	r := router.SetupRouter(h,authHandler)

	// 2. サーバー起動
	fmt.Println("🚀 APIサーバー起動: http://localhost:8080")
	r.Run(":8080")
}


func getEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		// ログを出してプログラムを終了させる（これが安全！）
		log.Fatalf("❌ 致命的エラー: 必須環境変数 '%s' が設定されていない！ .envファイルを確認するか、exportコマンドで設定する", key)
	}
	return value
}
