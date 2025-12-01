package main

import (
	"encoding/json"
	"io"
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
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 登録エンドポイント
	r.POST("/api/occurrences", createOccurrence)
    	r.GET("/api/occurrences", getOccurrences)       // 一覧
    	r.GET("/api/occurrences/:id", getOccurrenceDetail) // 詳細
	r.PUT("/api/occurrences/:id", updateOccurrence)   // 更新
	r.DELETE("/api/occurrences/:id", deleteOccurrence) // 削除

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
// ---------------------------------------------------------
// 3. 一覧取得ハンドラー
// ---------------------------------------------------------
func getOccurrences(c *gin.Context) {
	// 最新100件を取得するSPARQL
	query := `
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		
		SELECT ?id ?taxonName ?remarks
		WHERE {
			?id a dwc:Occurrence ;
				dwc:scientificName ?taxonName .
			OPTIONAL { ?id dwc:occurrenceRemarks ?remarks }
		}
		LIMIT 100
	`

	results, err := queryFuseki(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// フロントが使いやすい形に整形
	type ListItem struct {
		ID        string `json:"id"`
		TaxonName string `json:"taxon_label"`
		Remarks   string `json:"remarks"`
	}
	var list []ListItem

	for _, binding := range results {
		list = append(list, ListItem{
			ID:        binding["id"].Value,
			TaxonName: binding["taxonName"].Value,
			Remarks:   binding["remarks"].Value, // OPTIONALなので空文字かも
		})
	}

	c.JSON(http.StatusOK, list)
}

// ---------------------------------------------------------
// 4. 詳細取得ハンドラー
// ---------------------------------------------------------
func getOccurrenceDetail(c *gin.Context) {
	// URLパラメータからIDを取得 (例: uuid-1234...)
	idParam := c.Param("id")
	targetURI := "http://my-db.org/occ/" + idParam

	// そのIDの情報を全部取るSPARQL
	// ※形質(Trait)は複数あるのでGROUP_CONCATでまとめる手もあるけど、
	//   今回はシンプルにフラットに取ってGo側でまとめるのだ。
	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

		SELECT ?taxonName ?remarks ?traitID ?traitLabel
		WHERE {
			<%s> dwc:scientificName ?taxonName .
			OPTIONAL { <%s> dwc:occurrenceRemarks ?remarks }
			
			# 形質データ (あれば取得)
			OPTIONAL {
				<%s> ro:0000053 ?traitID .
				OPTIONAL { ?traitID rdfs:label ?traitLabel }
			}
		}
	`, targetURI, targetURI, targetURI)

	results, err := queryFuseki(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "データが見つからないのだ"})
		return
	}

	// データを整形 (Traitsを配列にまとめる)
	type DetailResp struct {
		ID         string  `json:"id"`
		TaxonName  string  `json:"taxon_label"`
		Remarks    string  `json:"remarks"`
		Traits     []Trait `json:"traits"`
	}
	
	// 最初の1行目から基本情報を取る
	resp := DetailResp{
		ID:        targetURI,
		TaxonName: results[0]["taxonName"].Value,
		Remarks:   results[0]["remarks"].Value,
		Traits:    []Trait{},
	}

	// 全行ループして形質リストを作る
	for _, b := range results {
		if tID, ok := b["traitID"]; ok {
			tLabel := ""
			if l, ok := b["traitLabel"]; ok {
				tLabel = l.Value
			}
			resp.Traits = append(resp.Traits, Trait{
				ID:    tID.Value,
				Label: tLabel,
			})
		}
	}

	c.JSON(http.StatusOK, resp)
}
// 共通: FusekiにSELECTクエリを投げる
func queryFuseki(sparql string) ([]map[string]BindingValue, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// GETリクエストでクエリを送信
	req, err := http.NewRequest("GET", "http://localhost:3030/biodb/query", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("query", sparql)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/sparql-results+json")
	// 認証が必要ならセット
	// req.SetBasicAuth(FusekiUser, FusekiPass) 

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	// JSONパース
	var result struct {
		Results struct {
			Bindings []map[string]BindingValue `json:"bindings"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results.Bindings, nil
}

// BindingValue構造体は既存のままでOK
type BindingValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
// ---------------------------------------------------------
// 6. 削除ハンドラー
// ---------------------------------------------------------
func deleteOccurrence(c *gin.Context) {
	idParam := c.Param("id")
	targetURI := "http://my-db.org/occ/" + idParam

	// そのURIを主語とするすべてのトリプルを削除
	sparql := fmt.Sprintf(`
		DELETE WHERE {
			<%s> ?p ?o .
		}
	`, targetURI)

	if err := sendToFuseki(sparql); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "削除失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "削除成功"})
}

// ---------------------------------------------------------
// 7. 更新ハンドラー
// ---------------------------------------------------------
func updateOccurrence(c *gin.Context) {
	idParam := c.Param("id")
	targetURI := "http://my-db.org/occ/" + idParam

	var req OccurrenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新用SPARQL: 古いデータを消して、新しいデータを入れる
	// DELETE WHERE { <URI> ?p ?o } ; INSERT DATA { ... }
	
	// まずはINSERTパートを作る（登録ロジックを再利用！）
	// ※ buildSPARQLはINSERT DATA { ... } 全体を作っちゃうので、
	//    中身だけ欲しいけど、今回は簡易的に「削除実行 -> 登録実行」の2ステップでやるのが安全なのだ。

	// 1. まず削除
	deleteSparql := fmt.Sprintf("DELETE WHERE { <%s> ?p ?o }", targetURI)
	if err := sendToFuseki(deleteSparql); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新前の削除失敗"})
		return
	}

	// 2. 新しいデータで登録（IDは既存のものを使う）
	insertSparql, err := buildSPARQL(targetURI, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "RDF生成エラー"})
		return
	}
	
	if err := sendToFuseki(insertSparql); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新データの保存失敗"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功", "id": targetURI})
}
