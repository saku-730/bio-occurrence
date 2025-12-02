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

	pgDBConn := infrastructure.NewPostgresDB(PGHost, PGPort, PGUser, PGPass, PGDB)

	// 2. 依存関係の組み立て (DI)
	// リポジトリ
	occRepo := repository.NewOccurrenceRepository(fusekiURL, fusekiUser, fusekiPass)
	searchRepo := repository.NewSearchRepository(meiliURL, meiliKey)
	userRepo := repository.NewUserRepository(pgDBConn)

	// サービス (★ここで userRepo を渡すのが重要！)
	occSvc := service.NewOccurrenceService(occRepo, searchRepo, userRepo)
	userSvc := service.NewUserService(userRepo)

	// ハンドラー
	occHandler := handler.NewOccurrenceHandler(occSvc)
	userHandler := handler.NewUserHandler(userSvc)

	// 3. ルーターセットアップ
	r := router.SetupRouter(occHandler, userHandler)

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
