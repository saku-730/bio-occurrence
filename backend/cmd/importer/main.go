package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

// Meilisearchの設定
const (
	MeiliURL    = "http://localhost:7700"
	MeiliKey    = "masterKey123" // docker-composeで設定したやつ
	IndexName   = "ontology"
)

// Meilisearchに登録するデータの形（ドキュメント）
type TermDoc struct {
	ID    string   `json:"id"`       // PATO:0000014
	Label string   `json:"label"`    // 赤色 (jaを優先)
	En    string   `json:"en"`       // red
	Uri   string   `json:"uri"`      // http://...
}

func main() {
	fmt.Println("🐢 オントロジーインポーター起動なのだ！")

	// 1. Meilisearchクライアントの準備
	client := meilisearch.New(MeiliURL, meilisearch.WithAPIKey(MeiliKey))

	// インデックスを作成
	_, err := client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        IndexName,
		PrimaryKey: "id",
	})
	if err != nil {
		fmt.Println("⚠️ インデックスはすでにあるかも（無視してOK）:", err)
	}

	// 2. Turtleファイルを読み込む
	terms, err := parseTurtle("../../data/ontologies/test_pato.ttl")
	if err != nil {
		log.Fatal("ファイル読み込みエラー:", err)
	}
	fmt.Printf("📝 %d 件の用語\n", len(terms))

	// 3. Meilisearchに登録
	// ★ここを修正！ 第2引数に nil を追加したのだ
	task, err := client.Index(IndexName).AddDocuments(terms, nil)
	if err != nil {
		log.Fatal("登録エラー:", err)
	}

	fmt.Printf("送信完了！TaskUID: %d\n", task.TaskUID)
	fmt.Println("数秒後に http://localhost:7700 で検索できるようになる")
}

// 簡易Turtleパーサー
func parseTurtle(filePath string) ([]TermDoc, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var docs []TermDoc
	scanner := bufio.NewScanner(file)

	// 正規表現のコンパイル
	reID := regexp.MustCompile(`(pato:\d+)`)
	reLabelJa := regexp.MustCompile(`"(.*)"@ja`)
	reLabelEn := regexp.MustCompile(`"(.*)"@en`)
	
	currentDoc := TermDoc{}

	for scanner.Scan() {
		line := scanner.Text()

		// IDの検出
		if matches := reID.FindStringSubmatch(line); len(matches) > 0 {
			// 前のデータがあれば保存
			if currentDoc.ID != "" {
				docs = append(docs, currentDoc)
			}
			
			// IDに含まれるコロン(:)をアンダースコア(_)に置換
			rawID := matches[1]
			safeID := strings.ReplaceAll(rawID, ":", "_")

			currentDoc = TermDoc{
				ID:  safeID,
				Uri: "http://purl.obolibrary.org/obo/" + safeID,
			}
		}

		// 日本語ラベルの検出
		if matches := reLabelJa.FindStringSubmatch(line); len(matches) > 0 {
			currentDoc.Label = matches[1]
		}
		// 英語ラベルの検出
		if matches := reLabelEn.FindStringSubmatch(line); len(matches) > 0 {
			currentDoc.En = matches[1]
		}
	}
	// 最後の1件を追加
	if currentDoc.ID != "" {
		docs = append(docs, currentDoc)
	}

	return docs, scanner.Err()
}

