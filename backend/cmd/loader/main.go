package main

import (
	"bufio"
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
	BatchSize       = 1000 // 一度に送信するトリプル数
)

var ontologyConfig = map[string]string{
	"pato.obo":      "http://my-db.org/ontology/pato",
	"ro.obo":        "http://my-db.org/ontology/ro",
	"envo.obo":      "http://my-db.org/ontology/envo",
	"ncbitaxon.obo": "http://my-db.org/ontology/ncbitaxon",
}

func main() {
	log.Println("🚀 Starting OBO to RDF Loader (Fuseki)")

	for filename, graphURI := range ontologyConfig {
		filePath := filepath.Join("data", "ontologies", filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// patoなどはロード済みかもしれないので、ない場合はスキップでもOK
			// log.Printf("⚠️ File not found: %s (skipping)", filename)
			continue
		}

		log.Printf("📝 Loading %s into graph <%s>...", filename, graphURI)
		if err := processAndLoad(filePath, graphURI); err != nil {
			log.Fatalf("❌ Failed to load %s: %v", filename, err)
		}
	}
	log.Println("✅ All ontologies loaded successfully!")
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

	// バッチ送信ヘルパー
	sendBatch := func() error {
		if len(triples) == 0 {
			return nil
		}
		
		query := fmt.Sprintf("INSERT DATA { GRAPH <%s> { \n%s\n } }", graphURI, strings.Join(triples, "\n"))
		
		if err := sendSPARQL(query); err != nil {
			return err
		}
		triples = []string{}
		fmt.Print(".") 
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "[Term]" {
			currentID = ""
			continue
		}
		
		if strings.HasPrefix(line, "id: ") {
			rawID := strings.TrimPrefix(line, "id: ")
			if !strings.Contains(rawID, ":") { continue }
			
			safeID := strings.ReplaceAll(rawID, ":", "_")
			currentURI := OboPurlBase + safeID
			currentID = fmt.Sprintf("<%s>", currentURI)
			
			triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/1999/02/22-rdf-syntax-ns#type> <http://www.w3.org/2002/07/owl#Class> .", currentID))

		} else if currentID != "" {
			if strings.HasPrefix(line, "name: ") {
				name := strings.TrimPrefix(line, "name: ")
				// ★修正: 強力なエスケープ関数を使う
				name = escapeString(name)
				triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/2000/01/rdf-schema#label> \"%s\" .", currentID, name))
			
			} else if strings.HasPrefix(line, "is_a: ") {
				parts := strings.Split(line, " ")
				if len(parts) > 1 {
					parentRawID := parts[1]
					if strings.Contains(parentRawID, ":") {
						parentSafeID := strings.ReplaceAll(parentRawID, ":", "_")
						triples = append(triples, fmt.Sprintf("%s <http://www.w3.org/2000/01/rdf-schema#subClassOf> <%s%s> .", currentID, OboPurlBase, parentSafeID))
					}
				}
			
			} else if strings.HasPrefix(line, "synonym: ") {
				matches := reSynonym.FindStringSubmatch(line)
				if len(matches) > 1 {
					syn := matches[1]
					// ★修正: ここもエスケープ
					syn = escapeString(syn)
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
	fmt.Println(" Done.")
	return scanner.Err()
}

// ★追加: SPARQL文字列用のエスケープ処理
func escapeString(s string) string {
	// バックスラッシュを先にエスケープしないと、後で増殖するので注意
	s = strings.ReplaceAll(s, "\\", "\\\\") 
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// 改行コードなども念のため潰しておく
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func sendSPARQL(query string) error {
	req, err := http.NewRequest("POST", FusekiUpdateURL, strings.NewReader(query))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")
	req.SetBasicAuth(FusekiUser, FusekiPass)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// ★修正: エラーの内容（Body）を読み取って表示する
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}
