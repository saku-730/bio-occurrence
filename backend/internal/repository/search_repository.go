package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

type OccurrenceDocument struct {
	ID         string   `json:"id"`
	TaxonID    string   `json:"taxon_id"`
	TaxonLabel string   `json:"taxon_label"`
	Remarks    string   `json:"remarks"`
	Traits     []string `json:"traits"`
	OwnerID    string   `json:"owner_id"`
	OwnerName  string   `json:"owner_name"`
	IsPublic   bool     `json:"is_public"`
}

type SearchRepository interface {
	IndexOccurrence(req model.OccurrenceRequest, id string, ownerID string, ownerName string) error
	DeleteOccurrence(id string) error
	Search(query string, currentUserID string, targetTaxonID []string) ([]OccurrenceDocument, error)
}

type searchRepository struct {
	client    meilisearch.ServiceManager
	indexName string
}

func NewSearchRepository(url, key string) SearchRepository {
	client := meilisearch.New(url, meilisearch.WithAPIKey(key))
	indexName := "occurrences"

	// 1. フィルタ可能な属性の設定
	// taxon_id で絞り込むために、ここに追加が必要なのだ！
	filterAttributes := []string{"traits", "taxon_label", "is_public", "owner_id", "taxon_id"}
	
	// ライブラリのバージョンによっては []string をそのまま渡せるけど、既存コードに合わせて interface変換しているのだ
	convertedAttributes := make([]interface{}, len(filterAttributes))
	for i, v := range filterAttributes {
		convertedAttributes[i] = v
	}
	client.Index(indexName).UpdateFilterableAttributes(&convertedAttributes)
	
	// 2. ★検索対象（キーワード検索）の属性設定
	// ここを設定することで、query検索が taxon_label を無視して remarks と traits だけを見るようになるのだ
	searchableAttributes := []string{"remarks", "traits"}
	client.Index(indexName).UpdateSearchableAttributes(&searchableAttributes)

	// Primary Keyの設定
	client.Index(indexName).UpdateIndex(&meilisearch.UpdateIndexRequestParams{
		PrimaryKey: "id",
	})
	
	return &searchRepository{
		client:    client,
		indexName: indexName,
	}
}

func (r *searchRepository) IndexOccurrence(req model.OccurrenceRequest, uri string, ownerID, ownerName string) error {
	doc := OccurrenceDocument{
		ID:         getIDFromURI(uri),
		TaxonID:    req.TaxonID,
		TaxonLabel: req.TaxonLabel,
		Remarks:    req.Remarks,
		Traits:     make([]string, 0, len(req.Traits)*3),
		OwnerID:    ownerID,
		OwnerName:  ownerName,
		IsPublic:   req.IsPublic,
	}
	
	for _, t := range req.Traits {
		doc.Traits = append(doc.Traits, t.ValueLabel)
		doc.Traits = append(doc.Traits, t.PredicateLabel)
		doc.Traits = append(doc.Traits, fmt.Sprintf("%s: %s", t.PredicateLabel, t.ValueLabel))
	}

	_, err := r.client.Index(r.indexName).AddDocuments([]OccurrenceDocument{doc}, nil)
	if err != nil {
		return fmt.Errorf("meilisearch indexing failed: %w", err)
	}
	return nil
}

func (r *searchRepository) DeleteOccurrence(uri string) error {
	id := getIDFromURI(uri)
	_, err := r.client.Index(r.indexName).DeleteDocument(id)
	return err
}

func (r *searchRepository) Search(query string, currentUserID string, targetTaxonIDs []string) ([]OccurrenceDocument, error) {
	// フィルタリングロジック
	filter := "is_public = true"
	if currentUserID != "" {
		filter = fmt.Sprintf("(is_public = true OR owner_id = '%s')", currentUserID)
	}

	if len(targetTaxonIDs) > 0 {
		// IN ["ncbi:1", "ncbi:2", ...] の形式を作る
		// 文字列の配列を ' で囲んでカンマ区切りにする
		quotedIDs := make([]string, len(targetTaxonIDs))
		for i, id := range targetTaxonIDs {
			quotedIDs[i] = fmt.Sprintf("'%s'", id)
		}
		inFilter := fmt.Sprintf("taxon_id IN [%s]", strings.Join(quotedIDs, ", "))
		
		filter = fmt.Sprintf("%s AND %s", filter, inFilter)
	}

	// ログ出力（デバッグ用）
	fmt.Printf("🔎 Meili Filter: %s\n", filter)

	searchRes, err := r.client.Index(r.indexName).Search(query, &meilisearch.SearchRequest{
		Limit:  50,
		Filter: filter,
	})
	// fmt.Print(searchRes) // デバッグ用出力はコメントアウトしておいたのだ
	if err != nil {
		return nil, err
	}

	var docs []OccurrenceDocument
	
	for _, hit := range searchRes.Hits {
		data, err := json.Marshal(hit)
		if err != nil {
			continue
		}

		var doc OccurrenceDocument
		if err := json.Unmarshal(data, &doc); err != nil {
			continue
		}
		
		docs = append(docs, doc)
	}
	
	return docs, nil
}

func getIDFromURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[i+1:]
		}
	}
	return uri
}
