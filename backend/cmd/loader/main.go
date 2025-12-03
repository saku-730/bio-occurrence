package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	FusekiUpdateURL = "http://localhost:3030/biodb/update"
	FusekiUser      = "admin"
	FusekiPass      = "admin123"
	OboPurlBase     = "http://purl.obolibrary.org/obo/"
	BatchSize       = 1000
)

var ontologyConfig = map[string]string{
	"pato.obo":      "http://my-db.org/ontology/pato",
	"ro.obo":        "http://my-db.org/ontology/ro",
	"envo.obo":      "http://my-db.org/ontology/envo",
	"ncbitaxon.obo": "http://my-db.org/ontology/ncbitaxon",
}

func main() {
	log.Println("🚀 Starting OBO to RDF Loader (Fuseki)")

	if err := waitForFuseki(); err != nil {
		log.Fatalf("❌ Fuseki is not ready: %v", err)
	}

	for filename, graphURI := range ontologyConfig {
		filePath := filepath.Join("data", "ontologies", filename)
		
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// log.Printf("⚠️  File not found: %s (skipping)", filename)
			continue
		}

		log.Printf("📝 Loading %s into graph <%s>...", filename, graphURI)
		
		if err := clearGraph(graphURI); err != nil {
			log.Printf("⚠️  Failed to clear graph %s: %v", graphURI, err)
		}

		if err := processAndLoad(filePath, graphURI); err != nil {
			log.Printf("❌ Failed to load %s: %v", filename, err)
		} else {
			log.Printf("   -> ✅ Loaded %s successfully.", filename)
		}
	}
	
	log.Println("🎉 All tasks completed.")
}

func processAndLoad(filePath, graphURI string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var triples []string
	currentID := ""
	
	reSynonym := regexp.MustCompile(`^synonym: "([^"]+)"`)

	sendBatch := func() error {
		if len(triples) == 0 {
			return nil
		}
		// クエリ組み立て
		query := fmt.Sprintf("INSERT DATA { GRAPH <%s> { \n%s\n } }", graphURI, strings.Join(triples, "\n"))
		
		// 送信
		if err := sendSPARQL(query); err != nil {
			// ★エラー時にクエリの冒頭を表示してデバッグしやすくする
			log.Printf("🔥 Error Query Sample: %s...", query[:min(len(query), 500)])
			return err
		}
		triples = []string{}
		fmt.Print(".")
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// コメント除去 (単純な ! だとURL内の ! も消える可能性があるので注意だが、OBOでは行末コメントが主)
		// 安全のため、引用符の外側の ! だけ消すのが理想だが、簡易的に実装
		if idx := strings.Index(line, "!"); idx != -1 {
			// " がない、もしくは ! が " より後ろにある（閉じてる）場合はコメントとみなす簡易チェック
			if !strings.Contains(line, "\"") || strings.LastIndex(line, "\"") < idx {
				line = strings.TrimSpace(line[:idx])
			}
		}
		if line == "" { continue }

		if line == "[Term]" {
			currentID = ""
			continue
		}
		if line == "[Typedef]" {
			currentID = ""
			continue
		}
		
		// --- ID ---
		if strings.HasPrefix(line, "id: ") {
			rawID := strings.TrimPrefix(line, "id: ")
			rawID = strings.TrimSpace(rawID) // ★追加: 前後の空白除去

			// 不正な文字が含まれていたらスキップ (URLとして無効なもの)
			if strings.ContainsAny(rawID, " <>\"{}|\\^`") {
				// log.Printf("⚠️ Skipping invalid ID: %s", rawID)
				currentID = ""
				continue
			}
			if !strings.Contains(rawID, ":") { continue }

			safeID := strings.ReplaceAll(rawID, ":", "_")
			currentID = fmt.Sprintf("<%s%s>", OboPurlBase, safeID)
			
			triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/2002/07/owl#Class> .", currentID))

		} else if currentID != "" {
			// --- Name ---
			if strings.HasPrefix(line, "name: ") {
				name := strings.TrimPrefix(line, "name: ")
				name = escapeString(name)
				triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/2000/01/rdf-schema#label> \"%s\" .", currentID, name))
			
			// --- Is_a ---
			} else if strings.HasPrefix(line, "is_a: ") {
				parentRawID := strings.TrimPrefix(line, "is_a: ")
				parentRawID = strings.TrimSpace(parentRawID) // ★追加
				
				if strings.Contains(parentRawID, ":") && !strings.ContainsAny(parentRawID, " <>\"{}|\\^`") {
					parentSafeID := strings.ReplaceAll(parentRawID, ":", "_")
					triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/2000/01/rdf-schema#subClassOf> <%s%s> .", currentID, OboPurlBase, parentSafeID))
				}
			
			// --- Synonym ---
			} else if strings.HasPrefix(line, "synonym: ") {
				matches := reSynonym.FindStringSubmatch(line)
				if len(matches) > 1 {
					syn := escapeString(matches[1])
					triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/2004/02/skos/core#altLabel> \"%s\" .", currentID, syn))
				}
			}
		}

		if len(triples) >= BatchSize {
			if err := sendBatch(); err != nil {
				return err
			}
		}
	}

	if err := sendBatch(); err != nil {
		return err
	}
	fmt.Println()
	return scanner.Err()
}

func waitForFuseki() error {
	// ... (前回と同じなので省略可、そのまま使う) ...
	// もし消してしまっていたら再掲するので言ってね
	for i := 0; i < 10; i++ {
		resp, err := http.Get("http://localhost:3030")
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func clearGraph(graphURI string) error {
	query := fmt.Sprintf("CLEAR GRAPH <%s>", graphURI)
	return sendSPARQL(query)
}

func sendSPARQL(query string) error {
	req, err := http.NewRequest("POST", FusekiUpdateURL, strings.NewReader(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")
	auth := FusekiUser + ":" + FusekiPass
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
