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

// OBO files to process
var ontologies = []string{"pato.obo", "ro.obo", "envo.obo"}

func main() {
	log.Println("🚀 Starting OBO Ontology Indexer")

	// 1. Initialize Meilisearch Client
	client := meilisearch.New(MeiliURL, meilisearch.WithAPIKey(MeiliKey))

	// 2. Run Indexing
	if err := RunOboIndexer(client); err != nil {
		log.Fatalf("❌ Indexing failed: %v", err)
	}

	log.Println("✅ Indexing process completed successfully.")
}

// RunOboIndexer orchestrates the parsing and indexing process
func RunOboIndexer(client meilisearch.ServiceManager) error {
	// Configure Index
	// ★修正点1: UpdateIndex には構造体を渡す必要があるのだ
	_, err := client.Index(IndexName).UpdateIndex(&meilisearch.UpdateIndexRequestParams{
		PrimaryKey: "id",
	})
	if err != nil {
		// Index might not exist yet or already has this setting, just log info
		log.Println("ℹ️  Index configuration check:", err)
	}

	allTerms := []TermDoc{}

	for _, filename := range ontologies {
		filePath := filepath.Join("data", "ontologies", filename)
		log.Printf("📝 Parsing %s (OBO format)...", filename)

		terms, err := parseOboFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", filename, err)
		}
		
		log.Printf("   -> Extracted %d terms from %s.", len(terms), filename)
		allTerms = append(allTerms, terms...)
	}

	log.Printf("📦 Preparing to submit %d documents...", len(allTerms))

	if len(allTerms) > 0 {
		// Send to Meilisearch
		// ★修正点2: 第2引数に nil を追加したのだ
		task, err := client.Index(IndexName).AddDocuments(allTerms, nil)
		if err != nil {
			return fmt.Errorf("meilisearch submission error: %w", err)
		}
		log.Printf("🚀 Submitted! Task UID: %d", task.TaskUID)
	} else {
		log.Println("⚠️ No terms extracted. Check file paths and content.")
	}

	return nil
}

// ---------------------------------------------------
// OBO Parser Logic (Standard Library Only)
// ---------------------------------------------------

func parseOboFile(filePath string) ([]TermDoc, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var docs []TermDoc
	var currentDoc *TermDoc

	scanner := bufio.NewScanner(file)
	
	// Regex to extract synonym text: synonym: "TEXT" ...
	reSynonym := regexp.MustCompile(`^synonym: "([^"]+)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Start of a new Term block
		if line == "[Term]" {
			// Save previous doc if exists
			if currentDoc != nil {
				if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
				if currentDoc.ID != "" { // Only save if ID exists
					docs = append(docs, *currentDoc)
				}
			}
			// Start new doc
			currentDoc = &TermDoc{
				Synonyms: []string{},
			}
			continue
		}

		// Skip if we are not inside a Term block
		if currentDoc == nil {
			continue
		}

		// Parse attributes
		if strings.HasPrefix(line, "id: ") {
			// id: PATO:0000014
			rawID := strings.TrimPrefix(line, "id: ")
			
			// OBO files often contain ID-spaces we don't need (like "is_a"), check for colon
			if !strings.Contains(rawID, ":") { 
				continue 
			}

			// Normalize ID for Meilisearch (PATO:123 -> PATO_123)
			safeID := strings.ReplaceAll(rawID, ":", "_")
			
			currentDoc.ID = safeID
			currentDoc.Uri = OboPurlBase + safeID
			
			// Extract ontology name
			parts := strings.Split(safeID, "_")
			if len(parts) > 0 {
				currentDoc.Ontology = parts[0]
			}

		} else if strings.HasPrefix(line, "name: ") {
			// name: red
			name := strings.TrimPrefix(line, "name: ")
			currentDoc.Label = name
			currentDoc.En = name // OBO is English by default

		} else if strings.HasPrefix(line, "synonym: ") {
			// synonym: "crimson" EXACT []
			matches := reSynonym.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentDoc.Synonyms = append(currentDoc.Synonyms, matches[1])
			}
		}
	}

	// Append the very last document
	if currentDoc != nil {
		if currentDoc.Label == "" { currentDoc.Label = currentDoc.En }
		if currentDoc.ID != "" {
			docs = append(docs, *currentDoc)
		}
	}

	return docs, scanner.Err()
}
