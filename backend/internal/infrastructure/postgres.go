package infrastructure

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func NewPostgresDB(host, port, user, password, dbname string) *sql.DB {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Postgres: %v", err)
	}

	// 接続確認（Ping）
	// コンテナ起動直後は接続できないことがあるので、数回リトライするロジックを入れると親切だけど、
	// 今回はシンプルにPingしてダメなら落ちるようにするのだ
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Postgres Ping failed: %v", err)
	}

	// 接続プールの設定（おまじない）
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db
}
