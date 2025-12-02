package repository

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"fmt"
	"encoding/json"

	"github.com/meilisearch/meilisearch-go"
)

// Meilisearchに登録するドキュメントの形
type OccurrenceDocument struct {
	ID         string   `json:"id"`
	TaxonID    string   `json:"taxon_id"`
	TaxonLabel string   `json:"taxon_label"`
	Remarks    string   `json:"remarks"`
	Traits     []string `json:"traits"`
	OwnerID    string   `json:"owner_id"`
	OwnerName  string   `json:"owner_name"`
}

type SearchRepository interface {
	IndexOccurrence(req model.OccurrenceRequest, id string, ownerID string, ownerName string) error
	DeleteOccurrence(id string) error
	Search(query string) ([]OccurrenceDocument, error)
}

type searchRepository struct {
	client    meilisearch.ServiceManager
	indexName string
}

func NewSearchRepository(url, key string) SearchRepository {
	client := meilisearch.New(url, meilisearch.WithAPIKey(key))
	indexName := "occurrences"

	// ★修正点1: プライマリキーを "id" に設定する処理を追加
	// これをやらないと、id と taxon_id で迷ってエラーになるのだ
	_, err := client.Index(indexName).UpdateIndex(&meilisearch.UpdateIndexRequestParams{
		PrimaryKey: "id",
	})
	if err != nil {
		// インデックスがまだない場合などはエラーになることもあるが、ログに出すだけでOK
		fmt.Printf("⚠️ Index update info: %v\n", err)
	}

	// フィルタリング設定（既存のコード）
	filterAttributes := []string{"traits", "taxon_label"}
	convertedAttributes := make([]interface{}, len(filterAttributes))
	for i, v := range filterAttributes {
		convertedAttributes[i] = v
	}

	client.Index(indexName).UpdateFilterableAttributes(&convertedAttributes)
	
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
		OwnerID:    ownerID,   // ★セット
		OwnerName:  ownerName, // ★セット
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

func (r *searchRepository) Search(query string) ([]OccurrenceDocument, error) {
	// 検索実行
	searchRes, err := r.client.Index(r.indexName).Search(query, &meilisearch.SearchRequest{
		Limit: 20,
	})
	if err != nil {
		return nil, err
	}

	var docs []OccurrenceDocument
	
	// ★修正箇所: スマートな変換ロジック
	for _, hit := range searchRes.Hits {
		// 1. hit (interface{}) を一度 JSONバイト列に戻す
		// (これで hit が map でも RawMessage でも関係なくなる！)
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

// ヘルパー: URIからID抽出
func getIDFromURI(uri string) string {
	for i := len(uri) - 1; i >= 0; i-- {
		if uri[i] == '/' {
			return uri[i+1:]
		}
	}
	return uri
}
