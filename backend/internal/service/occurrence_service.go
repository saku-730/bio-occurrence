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
}

type occurrenceService struct {
	repo repository.OccurrenceRepository
}

func NewOccurrenceService(repo repository.OccurrenceRepository) OccurrenceService {
	return &occurrenceService{repo: repo}
}

func (s *occurrenceService) Register(req model.OccurrenceRequest) (string, error) {
	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	err := s.repo.Create(occURI, req)
	return occURI, err
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
	return s.repo.Update(targetURI, req)
}

func (s *occurrenceService) Remove(id string) error {
	targetURI := "http://my-db.org/occ/" + id
	return s.repo.Delete(targetURI)
}

func (s *occurrenceService) GetTaxonStats(rawID string) (*model.TaxonStats, error) {
	safeID := strings.ReplaceAll(rawID, ":", "_")
	taxonURI := "http://purl.obolibrary.org/obo/" + safeID
	return s.repo.GetTaxonStats(taxonURI, rawID)
}
