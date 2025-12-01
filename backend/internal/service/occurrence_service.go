package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"strings"

	"github.com/google/uuid"
)

type OccurrenceService interface {
	Register(req model.OccurrenceRequest) (string, error)
	GetAll() ([]model.OccurrenceListItem, error)
	GetDetail(id string) (*model.OccurrenceDetail, error)
	Modify(id string, req model.OccurrenceRequest) error
	Remove(id string) error
	GetTaxonStats(rawID string) (*model.TaxonStats, error)
	Search(query string) ([]repository.OccurrenceDocument, error)
}

type occurrenceService struct {
	repo repository.OccurrenceRepository
	searchRepo repository.SearchRepository
}

func NewOccurrenceService(
	repo repository.OccurrenceRepository,
	searchRepo repository.SearchRepository,
) OccurrenceService {
	return &occurrenceService{
		repo:       repo,
		searchRepo: searchRepo,
	}
}


func (s *occurrenceService) Register(req model.OccurrenceRequest) (string, error) {
	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	
	// 1. Fusekiに保存 (これが正)
	err := s.repo.Create(occURI, req)
	if err != nil {
		return "", err
	}

	// 2. Meilisearchにも保存 (検索用インデックス)
	// ※ここが失敗してもエラーにはせず、ログ出力にとどめる設計もあるが、今回はエラーを返す
	if err := s.searchRepo.IndexOccurrence(req, occURI); err != nil {
		// 本当はFusekiをロールバックするか、非同期でリトライすべきだけど簡易実装
		return occURI, err 
	}

	return occURI, nil
}

func (s *occurrenceService) GetAll() ([]model.OccurrenceListItem, error) {
	return s.repo.FindAll()
}

func (s *occurrenceService) GetDetail(id string) (*model.OccurrenceDetail, error) {
	// IDからURIを復元 (本来はIDのバリデーションなどをここでする)
	targetURI := "http://my-db.org/occ/" + id
	return s.repo.FindByID(targetURI)
}

func (s *occurrenceService) Modify(id string, req model.OccurrenceRequest) error {
	targetURI := "http://my-db.org/occ/" + id
	
	// 1. Fuseki更新
	if err := s.repo.Update(targetURI, req); err != nil {
		return err
	}
	
	// 2. Meilisearch更新 (上書き)
	return s.searchRepo.IndexOccurrence(req, targetURI)
}

func (s *occurrenceService) Remove(id string) error {
	targetURI := "http://my-db.org/occ/" + id
	
	// 1. Fuseki削除
	if err := s.repo.Delete(targetURI); err != nil {
		return err
	}
	
	// 2. Meilisearch削除
	return s.searchRepo.DeleteOccurrence(targetURI)
}

func (s *occurrenceService) GetTaxonStats(rawID string) (*model.TaxonStats, error) {
	safeID := strings.ReplaceAll(rawID, ":", "_")
	taxonURI := "http://purl.obolibrary.org/obo/" + safeID
	return s.repo.GetTaxonStats(taxonURI, rawID)
}

func (s *occurrenceService) Search(query string) ([]repository.OccurrenceDocument, error) {
	return s.searchRepo.Search(query)
}

