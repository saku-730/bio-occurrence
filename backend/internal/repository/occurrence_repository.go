package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"
)

type OccurrenceRepository interface {
	Create(uri string, userID string, req model.OccurrenceRequest) error
	FindAll(currentUserID string) ([]model.OccurrenceListItem, error)
	FindByID(uri string) (*model.OccurrenceDetail, error)
	Update(uri string, userID string, req model.OccurrenceRequest) error
	Delete(uri string) error
	GetTaxonStats(taxonURI string, rawID string) (*model.TaxonStats, error)
	GetDescendantIDs(label string) ([]string, error)
	GetTaxonIDByLabel(label string) (string, error)
}

type occurrenceRepository struct {
	updateURL string
	queryURL  string
	username  string
	password  string
	client    *http.Client
}

func NewOccurrenceRepository(baseURL, user, pass string) OccurrenceRepository {
	return &occurrenceRepository{
		updateURL: baseURL + "/update",
		queryURL:  baseURL + "/query",
		username:  user,
		password:  pass,
		client:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (r *occurrenceRepository) Create(uri string, userID string, req model.OccurrenceRequest) error {
	sparql, err := r.buildInsertSPARQL(uri, userID, req)
	if err != nil {
		return err
	}
	return r.sendUpdate(sparql)
}

func (r *occurrenceRepository) FindAll(currentUserID string) ([]model.OccurrenceListItem, error) {
	filter := "(!BOUND(?vis) || ?vis = \"public\")"
	if currentUserID != "" {
		filter += fmt.Sprintf(" || (BOUND(?creator) && str(?creator) = \"http://my-db.org/user/%s\")", currentUserID)
	}

	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		PREFIX ex: <http://my-db.org/data/>
		
		SELECT ?id ?taxonName ?remarks ?creator ?created
		WHERE {
			?id a dwc:Occurrence ;
				dwc:scientificName ?taxonName .
			OPTIONAL { ?id dwc:occurrenceRemarks ?remarks }
			OPTIONAL { ?id dcterms:creator ?creator }
			OPTIONAL { ?id ex:visibility ?vis }
			OPTIONAL { ?id dcterms:created ?created }

			FILTER (%s)
		}
		ORDER BY DESC(?created)
		LIMIT 100
	`, filter)
	
	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}

	var list []model.OccurrenceListItem
	for _, b := range results {
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
			OwnerID:   ownerID,
			OwnerName: "",
			CreatedAt: safeValue(b, "created"),
		})
	}
	return list, nil
}

func (r *occurrenceRepository) FindByID(uri string) (*model.OccurrenceDetail, error) {
	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		PREFIX ex: <http://my-db.org/data/>

		SELECT ?taxonName ?remarks ?pred ?predLabel ?val ?valLabel ?creator ?vis ?created
		WHERE {
			<%s> dwc:scientificName ?taxonName .
			OPTIONAL { <%s> dwc:occurrenceRemarks ?remarks }
			OPTIONAL { <%s> dcterms:creator ?creator }
			OPTIONAL { <%s> ex:visibility ?vis }
			OPTIONAL { <%s> dcterms:created ?created }
			
			OPTIONAL {
				<%s> ?pred ?val .
				OPTIONAL { ?pred rdfs:label ?predLabel }
				OPTIONAL { ?val rdfs:label ?valLabel }
			}
		}
	`, uri, uri, uri, uri, uri, uri)

	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}

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
		OwnerID:   ownerID,
		CreatedAt: safeValue(results[0], "created"),
		Traits:    []model.Trait{},
	}

	ignoredPredicates := map[string]bool{
		"http://www.w3.org/1999/02/22-rdf-syntax-ns#type": true,
		"http://rs.tdwg.org/dwc/terms/scientificName": true,
		"http://rs.tdwg.org/dwc/terms/scientificNameID": true,
		"http://rs.tdwg.org/dwc/terms/occurrenceRemarks": true,
		"http://purl.org/dc/terms/creator": true,
		"http://purl.org/dc/terms/created": true,
		"http://my-db.org/data/visibility": true,
	}

	seen := make(map[string]bool)
	for _, b := range results {
		predURI := b["pred"].Value
		if predURI == "" || ignoredPredicates[predURI] {
			continue
		}

		valURI := b["val"].Value
		key := predURI + valURI

		if !seen[key] {
			detail.Traits = append(detail.Traits, model.Trait{
				PredicateID:    shortenID(predURI),
				PredicateLabel: safeValue(b, "predLabel"),
				ValueID:        shortenID(valURI),
				ValueLabel:     safeValue(b, "valLabel"),
			})
			seen[key] = true
		}
	}
	return detail, nil
}

func (r *occurrenceRepository) Update(uri string, userID string, req model.OccurrenceRequest) error {
	deleteSparql := fmt.Sprintf("DELETE WHERE { <%s> ?p ?o }", uri)
	if err := r.sendUpdate(deleteSparql); err != nil {
		return fmt.Errorf("failed to delete old data: %w", err)
	}
	
	sparql, err := r.buildInsertSPARQL(uri, userID, req)
	if err != nil {
		return err
	}
	return r.sendUpdate(sparql)
}

func (r *occurrenceRepository) Delete(uri string) error {
	sparql := fmt.Sprintf("DELETE WHERE { <%s> ?p ?o }", uri)
	return r.sendUpdate(sparql)
}

func (r *occurrenceRepository) GetTaxonStats(taxonURI string, rawID string) (*model.TaxonStats, error) {
	if strings.HasPrefix(rawID, "ncbi:") {
		taxonURI = resolveURI(rawID, "", "user_taxon")
	}

	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

		SELECT (COUNT(?occ) AS ?count) (GROUP_CONCAT(DISTINCT ?traitLabel; separator=",") AS ?traits)
		WHERE {
			?occ dwc:scientificNameID <%s> .
			OPTIONAL {
				?occ ?pred ?val .
				?val rdfs:label ?traitLabel .
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

// ★修正: 名前から子孫IDを取得 (推論検索用)
// label だけでなく altLabel (別名) も検索する！
func (r *occurrenceRepository) GetDescendantIDs(label string) ([]string, error) {
	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		
		SELECT DISTINCT (?uri AS ?id)
		WHERE {
		  # 1. オントロジーから「その名前の概念」と「子孫」を探す
		  GRAPH <http://my-db.org/ontology/ncbitaxon> {
			# ★修正: UNIONを使って別名も検索
			{ ?root rdfs:label ?name } UNION { ?root skos:altLabel ?name }
			FILTER (lcase(str(?name)) = lcase("%s"))
			
			?uri rdfs:subClassOf* ?root .
		  }

		  # 2. 実際にオカレンスデータで使われているIDだけに絞る
		  ?occ dwc:scientificNameID ?uri .
		}
		LIMIT 1000
	`, label)

	results, err := r.sendQuery(query)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, b := range results {
		if val, ok := b["id"]; ok {
			uri := val.Value
			if strings.Contains(uri, "NCBITaxon_") {
				parts := strings.Split(uri, "NCBITaxon_")
				if len(parts) > 1 {
					ids = append(ids, "ncbi:"+parts[1])
				}
			} else if strings.Contains(uri, "ncbi_") {
				parts := strings.Split(uri, "ncbi_")
				if len(parts) > 1 {
					ids = append(ids, "ncbi:"+parts[1])
				}
			}
		}
	}
	return ids, nil
}


// ★修正: 名前からIDを引く (検索用)
// label だけでなく altLabel も検索！
func (r *occurrenceRepository) GetTaxonIDByLabel(label string) (string, error) {
	log.Printf("🔍 [GetTaxonIDByLabel] Start looking up: '%s'", label)

	query := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		
		SELECT ?uri
		WHERE {
		  GRAPH <http://my-db.org/ontology/ncbitaxon> {
			{ ?uri rdfs:label ?name } UNION { ?uri skos:altLabel ?name }
			FILTER (lcase(str(?name)) = lcase("%s"))
		  }
		}
		LIMIT 1
	`, label)

	results, err := r.sendQuery(query)
	if err != nil {
		log.Printf("❌ [GetTaxonIDByLabel] Query failed: %v", err)
		return "", err
	}
	
	log.Printf("✅ [GetTaxonIDByLabel] Found %d results", len(results))

	if len(results) == 0 {
		log.Printf("   -> No ID found for '%s'", label)
		return "", nil
	}

	uri := safeValue(results[0], "uri")
	log.Printf("   -> Raw URI found: %s", uri)

	if strings.Contains(uri, "NCBITaxon_") {
		parts := strings.Split(uri, "NCBITaxon_")
		if len(parts) > 1 {
			id := "ncbi:" + parts[1]
			log.Printf("   -> Converted ID: %s", id)
			return id, nil
		}
	}
	return "", nil
}

// ---------------------------------------------------
// Helper
// ---------------------------------------------------

func (r *occurrenceRepository) buildInsertSPARQL(uri string, userID string, req model.OccurrenceRequest) (string, error) {
	visibility := "private"
	if req.IsPublic {
		visibility = "public"
	}
	
	taxonID := req.TaxonID
	if taxonID == "" { taxonID = "ncbi:unknown" }
	taxonLabel := req.TaxonLabel
	if taxonLabel == "" { taxonLabel = "未同定" }

	now := time.Now().Format(time.RFC3339)

	const tpl = `
PREFIX ex: <http://my-db.org/data/>
PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX ro: <http://purl.obolibrary.org/obo/RO_>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>

INSERT DATA {
  <{{.URI}}> 
    a dwc:Occurrence ;
    dwc:scientificNameID <{{.TaxonURI}}> ;
    dwc:scientificName "{{.TaxonLabel}}" ;
    dcterms:creator <http://my-db.org/user/{{.UserID}}> ;
    ex:visibility "{{.Visibility}}" ;
    dcterms:created "{{.CreatedAt}}"^^xsd:dateTime ;
    dwc:occurrenceRemarks "{{.Remarks}}" .

  {{range .Traits}}
  <{{$.URI}}> <{{.PredURI}}> <{{.ValURI}}> .
  <{{.PredURI}}> rdfs:label "{{.PredLabel}}" .
  <{{.ValURI}}> rdfs:label "{{.ValLabel}}" .
  {{end}}
}
`
	type TraitSafe struct {
		PredURI, PredLabel, ValURI, ValLabel string
	}

	var safeTraits []TraitSafe
	for _, t := range req.Traits {
		safeTraits = append(safeTraits, TraitSafe{
			PredURI:   resolveURI(t.PredicateID, t.PredicateLabel, "user_prop"),
			PredLabel: t.PredicateLabel,
			ValURI:    resolveURI(t.ValueID, t.ValueLabel, "user_val"),
			ValLabel:  t.ValueLabel,
		})
	}

	data := struct {
		URI, TaxonURI, TaxonLabel, Remarks, UserID, Visibility, CreatedAt string
		Traits                                                            []TraitSafe
	}{
		URI:        uri,
		TaxonURI:   resolveURI(taxonID, taxonLabel, "user_taxon"),
		TaxonLabel: taxonLabel,
		Remarks:    req.Remarks,
		UserID:     userID,
		Visibility: visibility,
		CreatedAt:  now,
		Traits:     safeTraits,
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

func resolveURI(id, label, userType string) string {
	if id != "" {
		if strings.HasPrefix(id, "http") { return id }
		safeID := strings.ReplaceAll(id, ":", "_")
		if strings.HasPrefix(id, "ncbi:") {
			safeID = strings.Replace(safeID, "ncbi_", "NCBITaxon_", 1)
		}
		return "http://purl.obolibrary.org/obo/" + safeID
	}
	encodedLabel := url.PathEscape(label)
	return fmt.Sprintf("http://my-db.org/%s/%s", userType, encodedLabel)
}

func shortenID(uri string) string {
	if strings.Contains(uri, "/obo/") {
		parts := strings.Split(uri, "/obo/")
		id := parts[len(parts)-1]
		id = strings.Replace(id, "NCBITaxon_", "ncbi:", 1)
		return strings.ReplaceAll(id, "_", ":")
	}
	return uri
}

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

func (r *occurrenceRepository) sendQuery(sparql string) ([]map[string]bindingValue, error) {
	data := url.Values{}
	data.Set("query", sparql)

	// ★追加: クエリの内容を表示
	log.Printf("📡 [sendQuery] Sending SPARQL:\n%s", sparql)

	req, err := http.NewRequest("POST", r.queryURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	r.setBasicAuth(req)

	resp, err := r.client.Do(req)
	if err != nil {
		log.Printf("❌ [sendQuery] Request error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	
	// ★追加: レスポンスの中身（JSON）を表示
	log.Printf("📥 [sendQuery] Response Status: %s", resp.Status)
	log.Printf("📥 [sendQuery] Response Body: %s", string(bodyBytes))

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result sparqlResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		log.Printf("❌ [sendQuery] JSON Decode Error: %v", err)
		return nil, err
	}
	return result.Results.Bindings, nil
}

func (r *occurrenceRepository) setBasicAuth(req *http.Request) {
	auth := r.username + ":" + r.password
	encoded := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encoded)
}

type sparqlResponse struct {
	Results struct {
		Bindings []map[string]bindingValue `json:"bindings"`
	} `json:"results"`
}
type bindingValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func safeValue(binding map[string]bindingValue, key string) string {
	if v, ok := binding[key]; ok {
		return v.Value
	}
	return ""
}
