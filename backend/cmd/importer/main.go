package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"io"

	// Meilisearch関連のimportは、Task Bで使うので一旦残しておくのだ
//	"github.com/meilisearch/meilisearch-go" 
)

// 設定 (main.goと共有するが、ここでは定義しておく)
const (
	FusekiUpdateURL = "http://localhost:3030/biodb/update"
	FusekiUser      = "admin"
	FusekiPass      = "admin123"
	
	// Dockerにマウントしたオントロジーファイルのパス（コンテナ内パス！）
	DockerOntologyPath = "/fuseki/data/ontologies"
)

// ロード対象のオントロジーリスト
var ontologies = []string{"pato.owl", "ro.owl", "envo.owl"}

func main() {
	fmt.Println("🚀 オントロジー知識ベースのロードを開始するのだ！")

	// Task A: Fusekiへのロード
	if err := LoadOntologies(); err != nil {
		log.Fatalf("❌ Fusekiへのロードに失敗したのだ: %v", err)
	}

	// Task B: Meilisearchの検索辞書作成（今は無視してOK）
	// if err := IndexOntologies(); err != nil {
	// 	log.Fatalf("❌ Meilisearchへの登録に失敗したのだ: %v", err)
	// }
    
	fmt.Println("✅ 全てのオントロジーをロード完了したのだ！")
}

// ---------------------------------------------------
// Task A: FusekiにLOAD命令を送るロジック
// ---------------------------------------------------

func LoadOntologies() error {
	for _, filename := range ontologies {
		// 1. ロード元と格納先のURIを定義
		// コンテナ内のファイルパスを指定するのだ (例: file:///fuseki/data/ontologies/pato.owl)
		fileURL := fmt.Sprintf("file://%s/%s", DockerOntologyPath, filename)
		fmt.Printf("file://%s/%s", DockerOntologyPath, filename)
		
		// グラフURIを定義 (例: http://my-db.org/ontology/pato)
		graphURI := fmt.Sprintf("http://my-db.org/ontology/%s", strings.TrimSuffix(filename, ".owl"))

		// 2. SPARQL LOAD コマンドを組み立てる
		sparqlUpdate := fmt.Sprintf("LOAD <%s> INTO GRAPH <%s>", fileURL, graphURI)

		fmt.Printf("  - ⏳ %s を %s にロード中...\n", filename, graphURI)
		
		// 3. Fusekiに送信
		if err := sendUpdate(sparqlUpdate); err != nil {
			return fmt.Errorf("failed to load %s: %w", filename, err)
		}
		fmt.Printf("  - ✅ %s ロード成功！\n", filename)
	}
	return nil
}

// ---------------------------------------------------
// 共通ヘルパー (Repositoryから移動)
// ---------------------------------------------------

// Fusekiへ更新リクエスト(POST)を送る
func sendUpdate(sparql string) error {
	req, err := http.NewRequest("POST", FusekiUpdateURL, strings.NewReader(sparql))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")
	req.SetBasicAuth(FusekiUser, FusekiPass) // 認証情報をセット

	client := &http.Client{Timeout: 60 * time.Second} // ★タイムアウトを長めにする (ファイルが大きいから！)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

