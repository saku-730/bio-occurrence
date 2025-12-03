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
	Ancestors  []string `json:"ancestors"`
}

type SearchRepository interface {
	IndexOccurrence(req model.OccurrenceRequest, id string, ownerID string, ownerName string, ancestors []string) error
	DeleteOccurrence(id string) error
	Search(query string, currentUserID string, targetTaxonID string) ([]OccurrenceDocument, error)
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
		Traits:     make([]string, 0, len(req.Traits)*3), // 少し多めに確保
		OwnerID:    ownerID,
		OwnerName:  ownerName,
		IsPublic:   req.IsPublic,
		Ancestors:  ancestors,
	}
	
	for _, t := range req.Traits {
		// 検索しやすいように色々なパターンで文字列化して入れる
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


func (r *searchRepository) Search(query string, currentUserID string, taxonIDs []string) ([]OccurrenceDocument, error) {
	// 1. 権限フィルター (公開 or 自分)
	filter := "is_public = true"
	if currentUserID != "" {
		filter = fmt.Sprintf("(is_public = true OR owner_id = '%s')", currentUserID)
	}

	if targetTaxonID != "" {
		// 「このデータの祖先リストの中に、指定されたIDが含まれているか？」
		// ancestors = 'ncbi:123' というフィルタで、配列内の要素との一致判定ができる
		taxonFilter := fmt.Sprintf("ancestors = '%s'", targetTaxonID)
		filter = fmt.Sprintf("(%s) AND (%s)", filter, taxonFilter)
		
		// 分類で絞り込んだ場合は、キーワード検索を空にする（全件表示）
		// ただし、キーワードが入力されている場合はAND検索にする
		// (今回は簡易的に、分類指定があればキーワードは無視する仕様にする)
		if query == "" { // queryが空でなければキーワードも活かす
        } else {
            // キーワード検索も併用したい場合はそのままでいいが、
            // "Vertebrata" というキーワード自体はヒットしないので空にするのが無難
            // query = "" 
        }
	}



	searchRes, err := r.client.Index(r.indexName).Search(query, &meilisearch.SearchRequest{
		Limit:  50,
		Filter: filter,
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
