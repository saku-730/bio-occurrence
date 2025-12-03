package model

// APIのリクエストで受け取るデータ
type OccurrenceRequest struct {
	TaxonID    string  `json:"taxon_id"`
	TaxonLabel string  `json:"taxon_label"`
	Traits     []Trait `json:"traits"`
	Remarks    string  `json:"remarks"`
	IsPublic   bool    `json:"is_public"`
}

// 形質データ (トリプル構造)
type Trait struct {
	// 述語 (Predicate)
	PredicateID    string `json:"predicate_id"`
	PredicateLabel string `json:"predicate_label"`

	// 値 (Object / Value)
	ValueID    string `json:"value_id"`
	ValueLabel string `json:"value_label"`
}

type OccurrenceListItem struct {
	ID        string `json:"id"`
	TaxonName string `json:"taxon_label"`
	Remarks   string `json:"remarks"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
	CreatedAt string `json:"created_at"`
}

type OccurrenceDetail struct {
	ID        string  `json:"id"`
	TaxonName string  `json:"taxon_label"`
	Remarks   string  `json:"remarks"`
	Traits    []Trait `json:"traits"`
	OwnerID   string  `json:"owner_id"`
	OwnerName string  `json:"owner_name"`
	CreatedAt string `json:"created_at"`
}

type TaxonStats struct {
	TaxonID    string   `json:"taxon_id"`
	TotalCount string   `json:"total_count"`
	Traits     []string `json:"traits"`
}
