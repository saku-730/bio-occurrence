package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

// Global Configuration
const (
	MeiliURL    = "http://localhost:7700"
	MeiliKey    = "masterKey123"
	IndexName   = "ontology"
	OboPurlBase = "http://purl.obolibrary.org/obo/"
	BatchSize   = 2000 // ★2000件ごとに送信する設定
)

// TermDoc struct for Meilisearch
type TermDoc struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	En       string   `json:"en"`
	Uri      string   `json:"uri"`
	Synonyms []string `json:"synonyms"`
	Ontology string   `json:"ontology"`
}

// ファイル名と登録先インデックスの対応表
var ontologyConfig = map[string]string{
	"pato.obo":      "ontology",
	"ro.obo":        "ontology",
	"envo.obo":      "ontology",
	"ncbitaxon.obo": "classification",
}

func main() {
	log.Println("🚀 Starting Multi-Index OBO Indexer")

	client := meilisearch.New(MeiliURL, meilisearch.WithAPIKey(MeiliKey))

	if err := RunBatchIndexer(client); err != nil {
		log.Fatalf("❌ Indexing failed: %v", err)
	}

	log.Println("✅ All indexing processes completed successfully.")
}

func RunBatchIndexer(client meilisearch.ServiceManager) error {
	// インデックス設定
	indices := []string{"ontology", "classification"}
	for _, idxName := range indices {
		client.Index(idxName).UpdateIndex(&meilisearch.UpdateIndexRequestParams{
			PrimaryKey: "id",
		})
		filterAttributes := []string{"ontology", "label", "id"}
		convertedAttributes := make([]interface{}, len(filterAttributes))
		for i, v := range filterAttributes {
			convertedAttributes[i] = v
		}
		client.Index(idxName).UpdateFilterableAttributes(&convertedAttributes)
		log.Printf("⚙️  Configured index: %s", idxName)
	}

	// ファイル処理
	for filename, targetIndex := range ontologyConfig {
		filePath := filepath.Join("data", "ontologies", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("⚠️  File not found: %s (skipping)", filename)
			continue
		}

		log.Printf("📝 Processing %s -> Index: [%s]", filename, targetIndex)
		count, err := processFileInBatches(client, filePath, targetIndex)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", filename, err)
		}
		log.Printf("   -> Finished %s. Total indexed: %d terms.", filename, count)
	}

	return nil
}

// ---------------------------------------------------
// Streaming OBO Parser & Batch Sender
// ---------------------------------------------------

func processFileInBatches(client meilisearch.ServiceManager, filePath, targetIndex string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 長い行に対応

	var batch []TermDoc
	var currentDoc *TermDoc
	totalCount := 0

	reSynonym := regexp.MustCompile(`^synonym: "([^"]+)"`)

	// バッチ送信ヘルパー
	sendBatch := func(docs []TermDoc) error {
		if len(docs) == 0 {
			return nil
		}
		_, err := client.Index(targetIndex).AddDocuments(docs, nil)
		if err != nil {
			return fmt.Errorf("meilisearch send error: %w", err)
		}
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// ★修正: [Term] だけでなく [Typedef] など全てのブロック開始を検知
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// 直前のドキュメントがあれば確定してバッチに追加
			if currentDoc != nil {
				if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
				// IDがある有効なデータのみ追加
				if currentDoc.ID != "" {
					batch = append(batch, *currentDoc)
				}
			}

			// バッチサイズを超えたら送信 (途中経過)
			if len(batch) >= BatchSize {
				if err := sendBatch(batch); err != nil {
					return totalCount, err
				}
				totalCount += len(batch)
				fmt.Printf("\r      ... Indexed %d terms", totalCount)
				batch = []TermDoc{} // バッチクリア
			}

			// 新しいブロックの開始
			if line == "[Term]" {
				currentDoc = &TermDoc{Synonyms: []string{}}
			} else {
				// [Typedef] など不要なブロックの場合は nil にしてスキップ
				currentDoc = nil
			}
			continue
		}

		// currentDocがない（Termブロック外）なら読み飛ばす
		if currentDoc == nil {
			continue
		}

		// 属性のパース
		if strings.HasPrefix(line, "id: ") {
			rawID := strings.TrimPrefix(line, "id: ")
			if !strings.Contains(rawID, ":") { continue }

			safeID := strings.ReplaceAll(rawID, ":", "_")
			currentDoc.ID = safeID
			currentDoc.Uri = OboPurlBase + safeID
			
			parts := strings.Split(safeID, "_")
			if len(parts) > 0 {
				currentDoc.Ontology = parts[0]
			}
		} else if strings.HasPrefix(line, "name: ") {
			name := strings.TrimPrefix(line, "name: ")
			currentDoc.Label = name
			currentDoc.En = name
		} else if strings.HasPrefix(line, "synonym: ") {
			matches := reSynonym.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentDoc.Synonyms = append(currentDoc.Synonyms, matches[1])
			}
		}
	}

	// ★重要: ループ終了後、最後の1件をバッチに追加
	if currentDoc != nil {
		if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
		if currentDoc.ID != "" {
			batch = append(batch, *currentDoc)
		}
	}

	// ★重要: バッチに残っている端数（例: 3200件中の200件）を送信
	log.Println("batch rest")
	if len(batch) > 0 {
		if err := sendBatch(batch); err != nil {
			return totalCount, err
		}
		totalCount += len(batch)
		fmt.Printf("\r      ... Indexed %d terms (Final flush)\n", totalCount)
	} else {
		fmt.Println() // 改行のみ
	}

	return totalCount, scanner.Err()
}
