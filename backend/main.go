package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Fuseki設定
const (
	FusekiUpdateURL = "http://localhost:3030/biodb/update"
	FusekiUser      = "admin"
	FusekiPass      = "admin123"
)

// 1. フロントから受け取るデータの形 (JSON)
type OccurrenceRequest struct {
	TaxonID    string   `json:"taxon_id" binding:"required"`    // 生物ID (例: ncbi:34844)
	TaxonLabel string   `json:"taxon_label" binding:"required"` // 生物名 (例: タヌキ)
	Traits     []Trait  `json:"traits"`                         // 形質リスト
	Remarks    string   `json:"remarks"`                        // 自由記述メモ
}

type Trait struct {
	ID    string `json:"id" binding:"required"`    // 形質ID (例: pato:0000014)
	Label string `json:"label" binding:"required"` // 形質名 (例: 赤色)
}

func main() {
	r := gin.Default()

	// CORS設定（Next.js:3000 からのアクセスを許可）
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 登録エンドポイント
	r.POST("/api/occurrences", createOccurrence)

	fmt.Println("🚀 APIサーバー起動: http://localhost:8080")
	r.Run(":8080")
}

// 2. 登録ハンドラー
func createOccurrence(c *gin.Context) {
	var req OccurrenceRequest
	// JSONのバリデーション
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// UUIDの発行 (オカレンスID)
	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID

	// 3. RDF (SPARQL Insert) の生成
	sparql, err := buildSPARQL(occURI, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "RDF変換エラー"})
		return
	}

	// 4. Fusekiに送信
	err = sendToFuseki(sparql)
	if err != nil {
		log.Printf("Fuseki Error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "データベース保存失敗"})
		return
	}

	// 成功レスポンス
	c.JSON(http.StatusCreated, gin.H{
		"message": "登録成功なのだ！",
		"id":      occURI,
	})
}

// RDF生成ロジック
func buildSPARQL(occURI string, req OccurrenceRequest) (string, error) {
	// SPARQLテンプレート
	const tpl = `
PREFIX ex: <http://my-db.org/data/>
PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

INSERT DATA {
  <{{.URI}}> 
    a dwc:Occurrence ;
    dwc:scientificNameID <http://purl.obolibrary.org/obo/{{.TaxonIDSafe}}> ;
    dwc:scientificName "{{.TaxonLabel}}" ;
    dwc:occurrenceRemarks "{{.Remarks}}" .

  {{range .Traits}}
  <{{$.URI}}> ro:0000053 <http://purl.obolibrary.org/obo/{{.IDSafe}}> .
  <http://purl.obolibrary.org/obo/{{.IDSafe}}> rdfs:label "{{.Label}}" .
  {{end}}
}
`
	// テンプレートにデータを埋め込む準備
	data := struct {
		URI         string
		TaxonIDSafe string
		TaxonLabel  string
		Remarks     string
		Traits      []struct{ IDSafe, Label string }
	}{
		URI:         occURI,
		TaxonIDSafe: strings.ReplaceAll(req.TaxonID, ":", "_"), // pato:123 -> pato_123
		TaxonLabel:  req.TaxonLabel,
		Remarks:     req.Remarks,
	}

	for _, t := range req.Traits {
		data.Traits = append(data.Traits, struct{ IDSafe, Label string }{
			IDSafe: strings.ReplaceAll(t.ID, ":", "_"),
			Label:  t.Label,
		})
	}

	t, err := template.New("sparql").Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Fuseki送信ロジック
func sendToFuseki(sparql string) error {
	req, err := http.NewRequest("POST", FusekiUpdateURL, strings.NewReader(sparql))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")
	req.SetBasicAuth(FusekiUser, FusekiPass)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	return nil
}
