package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"encoding/json"
	"fmt"

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
	Search(query string, currentUserID string) ([]OccurrenceDocument, error) // ★引数追加
}

type searchRepository struct {
	client    meilisearch.ServiceManager
	indexName string
}

func NewSearchRepository(url, key string) SearchRepository {
	client := meilisearch.New(url, meilisearch.WithAPIKey(key))
	indexName := "occurrences"

	// ★フィルタ用属性に is_public, owner_id を追加
	filterAttributes := []string{"traits", "taxon_label", "is_public", "owner_id"}
	convertedAttributes := make([]interface{}, len(filterAttributes))
	for i, v := range filterAttributes {
		convertedAttributes[i] = v
	}

	client.Index(indexName).UpdateFilterableAttributes(&convertedAttributes)
	
	// Primary Keyの設定も忘れずに
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
		Traits:     make([]string, len(req.Traits)),
		OwnerID:    ownerID,
		OwnerName:  ownerName,
		IsPublic:   req.IsPublic,
	}
	
	for i, t := range req.Traits {
		doc.Traits[i] = t.Label
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

func (r *searchRepository) Search(query string, currentUserID string) ([]OccurrenceDocument, error) {
	// フィルタリングロジック
	filter := "is_public = true"
	if currentUserID != "" {
		filter = fmt.Sprintf("(is_public = true OR owner_id = '%s')", currentUserID)
	}

	searchRes, err := r.client.Index(r.indexName).Search(query, &meilisearch.SearchRequest{
		Limit:  20,
		Filter: filter, // ★適用
	})
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
