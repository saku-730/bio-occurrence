package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
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
	BatchSize   = 2000
)

type TermDoc struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	En       string   `json:"en"`
	Ja       string   `json:"ja"`
	Uri      string   `json:"uri"`
	Synonyms []string `json:"synonyms"`
	Ontology string   `json:"ontology"`
}

// 設定: XSDを追加
var ontologyConfig = map[string]string{
//	"pato.obo":             "ontology",
//	"ro.obo":               "ontology",
//	"envo.obo":             "ontology",
//	"ncbitaxon.obo":        "classification",
	"tdwg_dwc_simple.xsd": "dwc", // ★XSDファイルを追加
}

func main() {
	log.Println("🚀 Starting Multi-Index Indexer (XSD Support)")

	client := meilisearch.New(MeiliURL, meilisearch.WithAPIKey(MeiliKey))

	if err := RunBatchIndexer(client); err != nil {
		log.Fatalf("❌ Indexing failed: %v", err)
	}

	log.Println("✅ All indexing processes completed successfully.")
}

func RunBatchIndexer(client meilisearch.ServiceManager) error {
	// インデックス初期設定
	indices := []string{"ontology", "classification", "dwc"}
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

	// ★修正: 日本語XML読み込み関数を定義して呼び出す
	dwcJaMap, err := loadJapanesXML("data/ontologies/tdwg_dwc_simple_ja.xsd")
	if err != nil {
		log.Printf("⚠️  Failed to load dwc_ja.xml: %v (continuing without JA)", err)
		dwcJaMap = make(map[string]string)
	} else {
		log.Printf("✅ Loaded %d Japanese terms from dwc_ja.xml", len(dwcJaMap))
	}

	// ファイル処理
	for filename, targetIndex := range ontologyConfig {
		filePath := filepath.Join("data", "ontologies", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// log.Printf("⚠️  File not found: %s (skipping)", filename)
			continue
		}

		log.Printf("📝 Processing %s -> Index: [%s]", filename, targetIndex)

		var count int
		var err error

		// 拡張子でパーサーを切り替え
		if strings.HasSuffix(filename, ".xsd") {
			// XSDパーサー (DwC用)
			// ★修正: dwcJaMap を渡すように変更
			count, err = processXsdFile(client, filePath, targetIndex, dwcJaMap)
		} else {
			// OBOパーサー (その他用)
			count, err = processFileInBatches(client, filePath, targetIndex)
		}

		if err != nil {
			return fmt.Errorf("failed to process %s: %w", filename, err)
		}
		log.Printf("   -> Finished %s. Total indexed: %d terms.", filename, count)
	}

	return nil
}

// ★追加: 日本語XML読み込み関数のスタブ
// 実際にはここでXMLをパースして、URI -> 日本語ラベル のマップを作るのだ
func loadJapanesXML(path string) (map[string]string, error) {
	// とりあえず空のマップを返す（エラー回避用）
	// 必要に応じて実装するのだ
	m := make(map[string]string)
	
	// 例: 手動でいくつか登録しておく場合
	// m["http://rs.tdwg.org/dwc/terms/occurrenceID"] = "オカレンスID"

	return m, nil
}

// ---------------------------------------------------
// XSD Parser for Darwin Core
// ---------------------------------------------------

// XSDの構造定義 (必要な部分のみ)
type XsElement struct {
	Ref string `xml:"ref,attr"`
}

type XsAll struct {
	Elements []XsElement `xml:"element"`
}

type XsComplexType struct {
	All XsAll `xml:"all"`
}

type XsSchema struct {
	Elements []struct {
		Name        string        `xml:"name,attr"`
		ComplexType XsComplexType `xml:"complexType"`
	} `xml:"element"`
}

// ★修正: jaMap 引数を追加
func processXsdFile(client meilisearch.ServiceManager, filePath, targetIndex string, jaMap map[string]string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	var schema XsSchema
	if err := xml.Unmarshal(byteValue, &schema); err != nil {
		return 0, fmt.Errorf("xml unmarshal error: %w", err)
	}

	var batch []TermDoc
	totalCount := 0

	// SimpleDarwinRecord の中身を探す
	for _, rootElem := range schema.Elements {
		if rootElem.Name == "SimpleDarwinRecord" {
			for _, elem := range rootElem.ComplexType.All.Elements {
				// ref="dwc:occurrenceID" のような形式
				ref := elem.Ref
				if ref == "" { continue }
				
				parts := strings.Split(ref, ":")
				if len(parts) != 2 { continue }
				
				prefix := parts[0]
				localName := parts[1]
				
				// ID生成
				safeID := prefix + "_" + localName // dwc_occurrenceID
				
				// ★修正: desc.Label は未定義なので localName を使う
				enLabel := localName

				// ★修正: URI生成ロジックを上に持ってきたのだ
				uri := ""
				if prefix == "dwc" {
					uri = "http://rs.tdwg.org/dwc/terms/" + localName
				} else if prefix == "dc" {
					uri = "http://purl.org/dc/elements/1.1/" + localName
				} else if prefix == "dcterms" {
					uri = "http://purl.org/dc/terms/" + localName
				}

				// ★修正: aboutURI は未定義なので、生成した uri を使ってマップから引く
				jaLabel := jaMap[uri]
				
				// 表示用ラベル
				displayLabel := enLabel
				if jaLabel != "" {
					displayLabel = jaLabel
				}

				doc := TermDoc{
					ID:       safeID,
					Label:    displayLabel,
					En:       enLabel,
					Ja:       jaLabel,
					Uri:      uri,
					Ontology: "DwC",
					Synonyms: []string{ref},
				}

				if jaLabel != "" {
					doc.Synonyms = append(doc.Synonyms, jaLabel)
				}

				batch = append(batch, doc)

				if len(batch) >= BatchSize {
					if _, err := client.Index(targetIndex).AddDocuments(batch, nil); err != nil {
						return totalCount, err
					}
					totalCount += len(batch)
					batch = []TermDoc{}
				}
			}
		}
	}

	// 残りを送信
	if len(batch) > 0 {
		if _, err := client.Index(targetIndex).AddDocuments(batch, nil); err != nil {
			return totalCount, err
		}
		totalCount += len(batch)
	}

	return totalCount, nil
}

// ---------------------------------------------------
// OBO Parser (Existing)
// ---------------------------------------------------
func processFileInBatches(client meilisearch.ServiceManager, filePath, targetIndex string) (int, error) {
	// ... (OBOパーサーの中身は変更なし、そのまま残す) ...
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var batch []TermDoc
	var currentDoc *TermDoc
	totalCount := 0

	reSynonym := regexp.MustCompile(`^synonym: "([^"]+)"`)

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

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if currentDoc != nil {
				if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
				if currentDoc.ID != "" {
					batch = append(batch, *currentDoc)
				}
			}

			if len(batch) >= BatchSize {
				if err := sendBatch(batch); err != nil {
					return totalCount, err
				}
				totalCount += len(batch)
				fmt.Printf("\r      ... Indexed %d terms", totalCount)
				batch = []TermDoc{}
			}

			if line == "[Term]" {
				currentDoc = &TermDoc{Synonyms: []string{}}
			} else {
				currentDoc = nil
			}
			continue
		}

		if currentDoc == nil { continue }

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

	if currentDoc != nil {
		if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
		if currentDoc.ID != "" {
			batch = append(batch, *currentDoc)
		}
	}
	if len(batch) > 0 {
		if err := sendBatch(batch); err != nil {
			return totalCount, err
		}
		totalCount += len(batch)
		fmt.Printf("\r      ... Indexed %d terms (Final flush)\n", totalCount)
	} else {
		fmt.Println()
	}

	return totalCount, scanner.Err()
}
