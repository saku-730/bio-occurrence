package model

// APIのリクエストで受け取るデータ
type OccurrenceRequest struct {
	TaxonID    string  `json:"taxon_id"`
	TaxonLabel string  `json:"taxon_label"`
	Traits     []Trait `json:"traits"`
	Remarks    string  `json:"remarks"`
	IsPublic   bool    `json:"is_public"`
}

type Trait struct {
	// 述語 (Predicate / Relationship)
	PredicateID    string `json:"predicate_id"`    // オントロジーID (例: ro:0002470) 。なければ空文字。
	PredicateLabel string `json:"predicate_label"` // 表示名 (例: 食べる)。必須。

	// 値 (Object / Value)
	ValueID    string `json:"value_id"`    // オントロジーID (例: ncbi:50557) 。なければ空文字。
	ValueLabel string `json:"value_label"` // 表示名 (例: 昆虫)。必須。
}

type OccurrenceListItem struct {
	ID        string `json:"id"`
	TaxonName string `json:"taxon_label"`
	Remarks   string `json:"remarks"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

// 詳細取得時のレスポンス
type OccurrenceDetail struct {
	ID        string  `json:"id"`
	TaxonName string  `json:"taxon_label"`
	Remarks   string  `json:"remarks"`
	Traits    []Trait `json:"traits"`
	OwnerID   string  `json:"owner_id"`
	OwnerName string  `json:"owner_name"`
}

// 種ごとの集計データのレスポンス
type TaxonStats struct {
	TaxonID    string   `json:"taxon_id"`
	TotalCount string   `json:"total_count"`
	Traits     []string `json:"traits"` // 簡易表示用に文字列リストのまま
}
