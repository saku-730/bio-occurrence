package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
//	"net/url"
	"strings"
	"text/template"
	"time"
)

// Repositoryのインターフェース
type OccurrenceRepository interface {
	Create(uri string, userID string, req model.OccurrenceRequest) error
	FindAll() ([]model.OccurrenceListItem, error)
	FindByID(uri string) (*model.OccurrenceDetail, error)
	Update(uri string, userID string, req model.OccurrenceRequest) error
	Delete(uri string) error
	GetTaxonStats(taxonURI string, rawID string) (*model.TaxonStats, error)
}

// 実装構造体
type occurrenceRepository struct {
	updateURL string
	queryURL  string
	username  string
	password  string
	client    *http.Client
}

// コンストラクタ
func NewOccurrenceRepository(baseURL, user, pass string) OccurrenceRepository {
	return &occurrenceRepository{
		updateURL: baseURL + "/update",
		queryURL:  baseURL + "/query",
		username:  user,
		password:  pass,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// ---------------------------------------------------
// CRUD実装
// ---------------------------------------------------

func (r *occurrenceRepository) Create(uri string,userID string, req model.OccurrenceRequest) error {
	sparql, err := r.buildInsertSPARQL(uri,userID, req)
	if err != nil {
		return err
	}
	return r.sendUpdate(sparql)
}

func (r *occurrenceRepository) FindAll() ([]model.OccurrenceListItem, error) {
	query := `
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		
		SELECT ?id ?taxonName ?remarks ?creator
		WHERE {
			?id a dwc:Occurrence ;
				dwc:scientificName ?taxonName .
			OPTIONAL { ?id dwc:occurrenceRemarks ?remarks }
			OPTIONAL { ?id dcterms:creator ?creator }
		}
		LIMIT 100
	`
	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}

	var list []model.OccurrenceListItem
	for _, b := range results {
		// URI (http://my-db.org/user/UUID) から UUID だけ抜く
		creatorURI := safeValue(b, "creator")
		ownerID := ""
		if creatorURI != "" {
			parts := strings.Split(creatorURI, "/")
			ownerID = parts[len(parts)-1]
		}

		list = append(list, model.OccurrenceListItem{
			ID:        b["id"].Value,
			TaxonName: b["taxonName"].Value,
			Remarks:   safeValue(b, "remarks"),
			OwnerID:   ownerID, // ★セット
			// OwnerName はここでは分からないので Service層で埋める
		})
	}
	return list, nil
}

// FindByID の修正
func (r *occurrenceRepository) FindByID(uri string) (*model.OccurrenceDetail, error) {
	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX dcterms: <http://purl.org/dc/terms/>

		SELECT ?taxonName ?remarks ?traitID ?traitLabel ?creator
		WHERE {
			<%s> dwc:scientificName ?taxonName .
			OPTIONAL { <%s> dwc:occurrenceRemarks ?remarks }
			OPTIONAL { <%s> dcterms:creator ?creator }
			OPTIONAL {
				<%s> ro:0000053 ?traitID .
				OPTIONAL { ?traitID rdfs:label ?traitLabel }
			}
		}
	`, uri, uri, uri, uri)

	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}

	// Creator IDの抽出
	creatorURI := safeValue(results[0], "creator")
	ownerID := ""
	if creatorURI != "" {
		parts := strings.Split(creatorURI, "/")
		ownerID = parts[len(parts)-1]
	}

	detail := &model.OccurrenceDetail{
		ID:        uri,
		TaxonName: results[0]["taxonName"].Value,
		Remarks:   safeValue(results[0], "remarks"),
		OwnerID:   ownerID, // ★セット
		Traits:    []model.Trait{},
	}

	seen := make(map[string]bool)
	for _, b := range results {
		if tID, ok := b["traitID"]; ok {
			if !seen[tID.Value] {
				detail.Traits = append(detail.Traits, model.Trait{
					ID:    tID.Value,
					Label: safeValue(b, "traitLabel"),
				})
				seen[tID.Value] = true
			}
		}
	}

	return detail, nil
}

func (r *occurrenceRepository) Update(uri string,userID string, req model.OccurrenceRequest) error {
	// 1. 削除
	deleteSparql := fmt.Sprintf("DELETE WHERE { <%s> ?p ?o }", uri)
	if err := r.sendUpdate(deleteSparql); err != nil {
		return fmt.Errorf("failed to delete old data: %w", err)
	}
	// 2. 登録
	insertSparql, err := r.buildInsertSPARQL(uri, userID, req)
	if err != nil {
		return err
	}
	return r.sendUpdate(insertSparql)
}

func (r *occurrenceRepository) Delete(uri string) error {
	sparql := fmt.Sprintf("DELETE WHERE { <%s> ?p ?o }", uri)
	return r.sendUpdate(sparql)
}

func (r *occurrenceRepository) GetTaxonStats(taxonURI string, rawID string) (*model.TaxonStats, error) {
	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

		SELECT (COUNT(?occ) AS ?count) (GROUP_CONCAT(DISTINCT ?traitLabel; separator=",") AS ?traits)
		WHERE {
			?occ dwc:scientificNameID <%s> .
			OPTIONAL {
				?occ ro:0000053 ?traitID .
				?traitID rdfs:label ?traitLabel .
			}
		}
	`, taxonURI)

	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}

	stats := &model.TaxonStats{TaxonID: rawID, TotalCount: "0", Traits: []string{}}
	if len(results) > 0 {
		stats.TotalCount = results[0]["count"].Value
		traitsStr := results[0]["traits"].Value
		if traitsStr != "" {
			stats.Traits = strings.Split(traitsStr, ",")
		}
	}
	return stats, nil
}

// ---------------------------------------------------
// 内部ヘルパー (Private)
// ---------------------------------------------------

// SPARQL INSERT文を組み立てる
func (r *occurrenceRepository) buildInsertSPARQL(uri string, userID string, req model.OccurrenceRequest) (string, error) {
	const tpl = `
PREFIX ex: <http://my-db.org/data/>
PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

INSERT DATA {
  <{{.URI}}> 
    a dwc:Occurrence ;
    dwc:scientificNameID <http://purl.obolibrary.org/obo/{{.TaxonIDSafe}}> ;
    dwc:scientificName "{{.TaxonLabel}}" ;
    dcterms:creator <http://my-db.org/user/{{.UserID}}> ;
    dwc:occurrenceRemarks "{{.Remarks}}" .

  {{range .Traits}}
  <{{$.URI}}> ro:0000053 <http://purl.obolibrary.org/obo/{{.IDSafe}}> .
  <http://purl.obolibrary.org/obo/{{.IDSafe}}> rdfs:label "{{.Label}}" .
  {{end}}
}
`
	type TraitSafe struct {
		IDSafe, Label string
	}
	data := struct {
		URI, TaxonIDSafe, TaxonLabel, Remarks, UserID string
		Traits                                []TraitSafe
	}{
		URI:         uri,
		TaxonIDSafe: strings.ReplaceAll(req.TaxonID, ":", "_"),
		TaxonLabel:  req.TaxonLabel,
		UserID: userID,
		Remarks:     req.Remarks,
	}
	for _, t := range req.Traits {
		data.Traits = append(data.Traits, TraitSafe{
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

// Fusekiへ更新リクエスト(POST)を送る
func (r *occurrenceRepository) sendUpdate(sparql string) error {
	req, err := http.NewRequest("POST", r.updateURL, strings.NewReader(sparql))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/sparql-update")
	r.setBasicAuth(req)

	resp, err := r.client.Do(req)
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

// Fusekiへ検索リクエスト(GET)を送る
func (r *occurrenceRepository) sendQuery(sparql string) ([]map[string]bindingValue, error) {
	req, err := http.NewRequest("GET", r.queryURL, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("query", sparql)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/sparql-results+json")
	r.setBasicAuth(req)

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result sparqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Results.Bindings, nil
}

func (r *occurrenceRepository) setBasicAuth(req *http.Request) {
	auth := r.username + ":" + r.password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)
}

// JSONパース用構造体
type sparqlResponse struct {
	Results struct {
		Bindings []map[string]bindingValue `json:"bindings"`
	} `json:"results"`
}
type bindingValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// 安全に値を取り出すヘルパー
func safeValue(binding map[string]bindingValue, key string) string {
	if v, ok := binding[key]; ok {
		return v.Value
	}
	return ""
}
