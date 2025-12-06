package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
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
	BatchSize       = 500 // 少し小さめに
)

var ontologyConfig = map[string]string{
	"pato.obo":      "http://my-db.org/ontology/pato",
	"ro.obo":        "http://my-db.org/ontology/ro",
	"envo.obo":      "http://my-db.org/ontology/envo",
	"ncbitaxon.obo": "http://my-db.org/ontology/ncbitaxon",
}

// 正規表現を事前コンパイル
var (
	reSynonym    = regexp.MustCompile(`^synonym:\s*"([^"]+)"`)
	reIDToken    = regexp.MustCompile(`^([A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+)`) // is_a などから最初の "PREFIX:ID" を抽出
	reIRIInvalid = regexp.MustCompile(`[ <>"{}|\\^` + "`" + `]`)
)

func main() {
	log.Println("🚀 Starting improved OBO to RDF Loader (Fuseki)")

	if err := waitForFuseki(); err != nil {
		log.Fatalf("❌ Fuseki is not ready: %v", err)
	}

	for filename, graphURI := range ontologyConfig {
		filePath := filepath.Join("data", "ontologies", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("⚠️  File not found: %s (skipping)", filePath)
			continue
		}

		log.Printf("📝 Loading %s into graph <%s>...", filename, graphURI)

		if err := clearGraphWithRetry(graphURI, 3); err != nil {
			log.Printf("⚠️  Failed to clear graph %s: %v", graphURI, err)
			// 続行するが注意を出す
		}

		if err := processAndLoad(filePath, graphURI); err != nil {
			log.Printf("❌ Failed to load %s: %v", filename, err)
		} else {
			log.Printf("   -> ✅ Loaded %s successfully.", filename)
		}
	}

	log.Println("🎉 All tasks completed.")
}

// processAndLoad: OBO をストリーミング処理し、Term 単位で重複を防ぎつつバッチで送信する
func processAndLoad(filePath, graphURI string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 128*1024)
	scanner.Buffer(buf, 4*1024*1024)

	var (
		triples          []string
		currentID        string
		currentTriples   []string
		seenSubjects     = make(map[string]struct{})
		sentBatches      = 0
		lineNo           = 0
	)

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		// コメント除去: 引用符内の ! は無視する
		line = stripCommentsPreserveQuotes(line)
		line = strings.TrimSpace(line)
		if line == "" { continue }

		if line == "[Term]" || line == "[Typedef]" {
			// 新しい Term ブロックの開始: 前の Term をフラッシュ
			if currentID != "" {
				if _, ok := seenSubjects[currentID]; !ok {
					triples = append(triples, currentTriples...)
					seenSubjects[currentID] = struct{}{}
				}

				currentTriples = currentTriples[:0]

				if len(triples) >= BatchSize {
					if err := sendBatch(triples, graphURI); err != nil {
						return fmt.Errorf("batch send failed at line %d: %w", lineNo, err)
					}
					triples = triples[:0]
					sentBatches++
					fmt.Print(".")
				}
			}
			currentID = ""
			continue
		}

		// --- id ---
		if strings.HasPrefix(line, "id:") {
			rawID := strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			rawID = stripSurroundingQuotes(rawID)
			// 不要な部分をカット
			if idx := strings.Index(rawID, " "); idx != -1 {
				rawID = rawID[:idx]
			}
			if idx := strings.Index(rawID, "["); idx != -1 {
				rawID = rawID[:idx]
			}
			rawID = strings.TrimSpace(rawID)

			if rawID == "" { currentID = ""; continue }
			if !strings.Contains(rawID, ":") { currentID = ""; continue }
			if reIRIInvalid.MatchString(rawID) {
				currentID = ""
				continue
			}

			safeID := strings.ReplaceAll(rawID, ":", "_")
			currentID = fmt.Sprintf("%s%s", OboPurlBase, safeID) // 文字列 URI

			// 型 triple を currentTriples に格納（後で一括送信）
			currentTriples = append(currentTriples, fmt.Sprintf("<%s> <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/2002/07/owl#Class> .", currentID))
			continue
		}

		// ここからは currentID が有効な場合のみ解析
		if currentID == "" { continue }

		// --- name ---
		if strings.HasPrefix(line, "name:") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = stripSurroundingQuotes(name)
			name = escapeString(name)
			currentTriples = append(currentTriples, fmt.Sprintf("<%s> <http://www.w3.org/2000/01/rdf-schema#label> \"%s\" .", currentID, name))
			continue
		}

		// --- is_a ---
		if strings.HasPrefix(line, "is_a:") {
			parentRaw := strings.TrimSpace(strings.TrimPrefix(line, "is_a:"))
			// 規則的に最初の TOKEN (PREFIX:ID) を抜き出す
			if m := reIDToken.FindStringSubmatch(parentRaw); len(m) > 1 {
				parent := m[1]
				parent = strings.TrimSpace(parent)
				if !reIRIInvalid.MatchString(parent) {
					parentSafe := strings.ReplaceAll(parent, ":", "_")
					currentTriples = append(currentTriples, fmt.Sprintf("<%s> <http://www.w3.org/2000/01/rdf-schema#subClassOf> <%s%s> .", currentID, OboPurlBase, parentSafe))
				}
			}
			continue
		}

		// --- synonym ---
		if strings.HasPrefix(line, "synonym:") {
			if m := reSynonym.FindStringSubmatch(line); len(m) > 1 {
				syn := escapeString(strings.TrimSpace(m[1]))
				currentTriples = append(currentTriples, fmt.Sprintf("<%s> <http://www.w3.org/2004/02/skos/core#altLabel> \"%s\" .", currentID, syn))
			}
			continue
		}

	}

	// 最後の Term をflush
	if currentID != "" {
		if _, ok := seenSubjects[currentID]; !ok {
			triples = append(triples, currentTriples...)
			seenSubjects[currentID] = struct{}{}
		}
	}

	// 残りを送る
	if len(triples) > 0 {
		if err := sendBatch(triples, graphURI); err != nil {
			return err
		}
	}

	fmt.Println()
	log.Printf("Sent %d batches", sentBatches)
	return scanner.Err()
}

// stripCommentsPreserveQuotes: 文字列中の " に挟まれた ! はコメント扱いにしない
func stripCommentsPreserveQuotes(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == '!' && !inQuote {
			// この位置で切る
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

func stripSurroundingQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1:len(s)-1]
	}
	return s
}

// sendBatch: triple を SPARQL Update に投げる
func sendBatch(triples []string, graphURI string) error {
	if len(triples) == 0 { return nil }
	query := fmt.Sprintf("INSERT DATA { GRAPH <%s> {\n%s\n} }", graphURI, strings.Join(triples, "\n"))
	return sendSPARQL(query)
}

func waitForFuseki() error {
	for i := 0; i < 30; i++ {
		resp, err := http.Get("http://localhost:3030")
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("fuseki not available")
}

// clearGraphWithRetry: CLEAR GRAPH をリトライする
func clearGraphWithRetry(graphURI string, retries int) error {
	for i := 0; i < retries; i++ {
		if err := sendSPARQL(fmt.Sprintf("CLEAR GRAPH <%s>", graphURI)); err != nil {
			log.Printf("clear graph failed (attempt %d/%d): %v", i+1, retries, err)
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("failed to clear graph after %d attempts", retries)
}

// sendSPARQL: HTTP POST で SPARQL Update を送る（Basic Auth）
func sendSPARQL(query string) error {
	req, err := http.NewRequest("POST", FusekiUpdateURL, strings.NewReader(query))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/sparql-update")
	auth := FusekiUser + ":" + FusekiPass
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// escapeString: RDF 文字列として安全にする
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// util: SHA1 を返す (デバッグ用に残してある)
func sha1hex(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
