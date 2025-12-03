package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"net/http"
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
		client:    &http.Client{Timeout: 10 * time.Second},
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
	// フィルタリングロジック: 公開 or 自分のデータ
	filter := "(!BOUND(?vis) || ?vis = \"public\")"
	if currentUserID != "" {
		filter += fmt.Sprintf(" || (BOUND(?creator) && str(?creator) = \"http://my-db.org/user/%s\")", currentUserID)
	}

	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		PREFIX ex: <http://my-db.org/data/>
		
		SELECT ?id ?taxonName ?remarks ?creator
		WHERE {
			?id a dwc:Occurrence ;
				dwc:scientificName ?taxonName .
			OPTIONAL { ?id dwc:occurrenceRemarks ?remarks }
			OPTIONAL { ?id dcterms:creator ?creator }
			OPTIONAL { ?id ex:visibility ?vis }

			FILTER (%s)
		}
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
		})
	}
	return list, nil
}


func (r *occurrenceRepository) FindByID(uri string) (*model.OccurrenceDetail, error) {
	query := fmt.Sprintf(`
		PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX dcterms: <http://purl.org/dc/terms/>
		PREFIX ex: <http://my-db.org/data/>

		SELECT ?taxonName ?remarks ?pred ?predLabel ?val ?valLabel ?creator ?vis
		WHERE {
			<%s> dwc:scientificName ?taxonName .
			OPTIONAL { <%s> dwc:occurrenceRemarks ?remarks }
			OPTIONAL { <%s> dcterms:creator ?creator }
			OPTIONAL { <%s> ex:visibility ?vis }
			
			# 形質データの取得 (述語 ?pred と 値 ?val)
			OPTIONAL {
				<%s> ?pred ?val .
				
				# 述語と値のラベルを取得 (なければURIそのものなどを出す)
				OPTIONAL { ?pred rdfs:label ?predLabel }
				OPTIONAL { ?val rdfs:label ?valLabel }
				
				# フィルタ: dwc: や dcterms: などのシステム用プロパティは除外したいが、
				# ここでは簡易的に全部取って、アプリ側でフィルタするか、
				# 登録時に専用のグラフに入れるのがベスト。
				# 今回は「ラベルがついているもの」を優先的に形質とみなすロジックにする。
			}
		}
	`, uri, uri, uri, uri, uri)

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
		Traits:    []model.Trait{},
	}

	ignoredPredicates := map[string]bool{
        "http://www.w3.org/1999/02/22-rdf-syntax-ns#type": true,
        "http://rs.tdwg.org/dwc/terms/scientificName": true,
        "http://rs.tdwg.org/dwc/terms/scientificNameID": true,
        "http://rs.tdwg.org/dwc/terms/occurrenceRemarks": true,
        "http://purl.org/dc/terms/creator": true,
        "http://my-db.org/data/visibility": true,
	}

	seen := make(map[string]bool)

	for _, b := range results {
	predURI := b["pred"].Value
	if ignoredPredicates[predURI] {
	    continue
	}

		valURI := b["val"].Value
		
		// ID生成 (簡易版)
		// http://purl.obolibrary.org/obo/RO_0002470 -> ro:0002470
	// http://my-db.org/user_prop/hoge -> user:hoge
		pID := shortenID(predURI)
		vID := shortenID(valURI)
		
		// 重複排除キー
		key := predURI + valURI
		if !seen[key] {
			detail.Traits = append(detail.Traits, model.Trait{
				PredicateID:    pID,
				PredicateLabel: safeValue(b, "predLabel"),
				ValueID:        vID,
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
// Helper
// ---------------------------------------------------

func (r *occurrenceRepository) buildInsertSPARQL(uri string, userID string, req model.OccurrenceRequest) (string, error) {
	// 公開設定の判定
	visibility := "private"
	if req.IsPublic {
		visibility = "public"
	}

	const tpl = `
PREFIX ex: <http://my-db.org/data/>
PREFIX dwc: <http://rs.tdwg.org/dwc/terms/>
PREFIX dcterms: <http://purl.org/dc/terms/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

INSERT DATA {
  <{{.URI}}> 
    a dwc:Occurrence ;
    dwc:scientificNameID <{{.TaxonURI}}> ;
    dwc:scientificName "{{.TaxonLabel}}" ;
    dcterms:creator <http://my-db.org/user/{{.UserID}}> ;
    ex:visibility "{{.Visibility}}" ;
    dwc:occurrenceRemarks "{{.Remarks}}" .

  {{range .Traits}}
  # <オカレンス> <述語> <値> .
  <{{$.URI}}> <{{.PredURI}}> <{{.ValURI}}> .
  
  # ラベル情報の保存 (表示用)
  <{{.PredURI}}> rdfs:label "{{.PredLabel}}" .
  <{{.ValURI}}> rdfs:label "{{.ValLabel}}" .
  {{end}}
}
`
	type TraitSafe struct {
		PredURI, PredLabel string
		ValURI, ValLabel   string
	}

	// URI生成ロジック
	var safeTraits []TraitSafe
	for _, t := range req.Traits {
		pURI := resolveURI(t.PredicateID, t.PredicateLabel, "user_prop")
		vURI := resolveURI(t.ValueID, t.ValueLabel, "user_val")
		
		safeTraits = append(safeTraits, TraitSafe{
			PredURI:   pURI,
			PredLabel: t.PredicateLabel,
			ValURI:    vURI,
			ValLabel:  t.ValueLabel,
		})
	}
    
    // TaxonURIも解決
    taxonURI := resolveURI(req.TaxonID, req.TaxonLabel, "user_taxon")
    // もしIDが空ならラベルを使うが、TaxonIDが空の場合は "ncbi:unknown" 扱いにするなどの処理も可

	data := struct {
		URI, TaxonURI, TaxonLabel, Remarks, UserID, Visibility string
		Traits                                                 []TraitSafe
	}{
		URI:        uri,
		TaxonURI:   taxonURI,
		TaxonLabel: req.TaxonLabel,
		Remarks:    req.Remarks,
		UserID:     userID,
		Visibility: visibility,
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

func resolveURI(id, label, userType string) string {
	if id != "" {
		// 既存のIDがある場合 (例: ro:0002470 -> http://purl.obolibrary.org/obo/ro_0002470)
        // すでにフルURIの場合はそのまま返す
        if strings.HasPrefix(id, "http") {
            return id
        }
		safeID := strings.ReplaceAll(id, ":", "_")
		return "http://purl.obolibrary.org/obo/" + safeID
	}
	
	// IDがない場合 (ユーザー独自入力): ラベルをURLエンコードしてID化
	// 例: "すごく赤い" -> http://my-db.org/user_val/%E3%81%99%E3%81%94...
	encodedLabel := url.PathEscape(label)
	return fmt.Sprintf("http://my-db.org/%s/%s", userType, encodedLabel)
}

func shortenID(uri string) string {
	// OBO形式
	if strings.Contains(uri, "/obo/") {
		parts := strings.Split(uri, "/obo/")
		return strings.ReplaceAll(parts[len(parts)-1], "_", ":")
	}
	// ユーザー定義
	if strings.Contains(uri, "my-db.org") {
        // そのまま返すか、適当に短縮
        return uri
    }
	return uri
}
