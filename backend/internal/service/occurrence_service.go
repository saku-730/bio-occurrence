package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"strings"
	"fmt"

	"github.com/google/uuid"
)

type OccurrenceService interface {
	Register(userID string, req model.OccurrenceRequest) (string, error)
	GetAll() ([]model.OccurrenceListItem, error)
	GetDetail(id string) (*model.OccurrenceDetail, error)
	Modify(userID string, id string, req model.OccurrenceRequest) error
	Remove(id string) error
	GetTaxonStats(rawID string) (*model.TaxonStats, error)
	Search(query string) ([]repository.OccurrenceDocument, error)
}

type occurrenceService struct {
	repo repository.OccurrenceRepository
	searchRepo repository.SearchRepository
	userRepo   repository.UserRepository
}

func NewOccurrenceService(
	repo repository.OccurrenceRepository,
	searchRepo repository.SearchRepository,
	userRepo repository.UserRepository,
) OccurrenceService {
	return &occurrenceService{
		repo:       repo,
		searchRepo: searchRepo,
		userRepo:   userRepo,
	}
}


func (s *occurrenceService) Register(userID string, req model.OccurrenceRequest) (string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	
	err = s.repo.Create(occURI, userID, req)
	if err != nil {
		return "", err
	}

	if err := s.searchRepo.IndexOccurrence(req, occURI, user.ID, user.Username); err != nil {
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

func (s *occurrenceService) Modify(userID string, id string, req model.OccurrenceRequest) error {
	targetURI := "http://my-db.org/occ/" + id
	
	// 1. Fuseki更新
	if err := s.repo.Update(targetURI,userID, req); err != nil {
		return err
	}
	
	// 2. Meilisearch更新 (上書き)
	return s.searchRepo.IndexOccurrence(req, targetURI)
}

func (s *occurrenceService) Modify(userID string, id string, req model.OccurrenceRequest) error {
	// 更新時もユーザー情報を再取得
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user")
	}

	targetURI := "http://my-db.org/occ/" + id
	
	if err := s.repo.Update(targetURI, userID, req); err != nil {
		return err
	}
	
	// Meilisearch更新
	return s.searchRepo.IndexOccurrence(req, targetURI, user.ID, user.Username)
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

