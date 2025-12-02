package model

// APIのリクエストで受け取るデータ
type OccurrenceRequest struct {
	TaxonID    string  `json:"taxon_id"`
	TaxonLabel string  `json:"taxon_label"`
	Traits     []Trait `json:"traits"`
	Remarks    string  `json:"remarks"`
}

// 形質データ
type Trait struct {
	ID    string `json:"id" binding:"required"`
	Label string `json:"label" binding:"required"`
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
	Traits     []string `json:"traits"`
}
