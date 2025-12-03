package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

type OccurrenceService interface {
	Register(userID string, req model.OccurrenceRequest) (string, error)
	GetAll(currentUserID string) ([]model.OccurrenceListItem, error)
	GetDetail(id string) (*model.OccurrenceDetail, error)
	Modify(userID string, id string, req model.OccurrenceRequest) error
	Remove(userID string, id string) error
	GetTaxonStats(rawID string) (*model.TaxonStats, error)
	Search(query string, currentUserID string) ([]repository.OccurrenceDocument, error)
}

type occurrenceService struct {
	repo       repository.OccurrenceRepository
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
	// 1. ユーザー情報を取得
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// 2. 祖先の取得 (Fusekiから)
	// これがないと検索時に引っかからないので重要！
	ancestors, err := s.repo.GetAncestorIDs(req.TaxonID)
	if err != nil {
		log.Printf("⚠️ 祖先の取得に失敗: %v", err)
		ancestors = []string{req.TaxonID} // 失敗しても自分自身は入れる
	}

	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	
	// 3. Fusekiに保存
	err = s.repo.Create(occURI, userID, req)
	if err != nil {
		return "", err
	}

	// 4. Meilisearchにも保存 (祖先リスト付きで！)
	if err := s.searchRepo.IndexOccurrence(req, occURI, user.ID, user.Username, ancestors); err != nil {
		return occURI, err 
	}

	return occURI, nil
}

func (s *occurrenceService) GetAll(currentUserID string) ([]model.OccurrenceListItem, error) {
	list, err := s.repo.FindAll(currentUserID)
	if err != nil {
		return nil, err
	}

	for i, item := range list {
		if item.OwnerID != "" {
			user, err := s.userRepo.FindByID(item.OwnerID)
			if err == nil && user != nil {
				list[i].OwnerName = user.Username
			} else {
				list[i].OwnerName = "Unknown"
			}
		}
	}
	return list, nil
}

func (s *occurrenceService) GetDetail(id string) (*model.OccurrenceDetail, error) {
	targetURI := "http://my-db.org/occ/" + id
	detail, err := s.repo.FindByID(targetURI)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}

	if detail.OwnerID != "" {
		user, err := s.userRepo.FindByID(detail.OwnerID)
		if err == nil && user != nil {
			detail.OwnerName = user.Username
		} else {
			detail.OwnerName = "Unknown"
		}
	}

	return detail, nil
}

func (s *occurrenceService) Modify(userID string, id string, req model.OccurrenceRequest) error {
	targetURI := "http://my-db.org/occ/" + id

	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}
	if existing.OwnerID != userID {
		return fmt.Errorf("permission denied: あなたのデータではないのだ")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user")
	}

	// 更新時も祖先を再取得
	ancestors, err := s.repo.GetAncestorIDs(req.TaxonID)
	if err != nil {
		log.Printf("⚠️ 祖先の取得に失敗: %v", err)
		ancestors = []string{req.TaxonID}
	}

	if err := s.repo.Update(targetURI, userID, req); err != nil {
		return err
	}
	
	return s.searchRepo.IndexOccurrence(req, targetURI, user.ID, user.Username, ancestors)
}

func (s *occurrenceService) Remove(userID string, id string) error {
	targetURI := "http://my-db.org/occ/" + id
	
	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}
	if existing.OwnerID != userID {
		return fmt.Errorf("permission denied: 他人のデータは消せないのだ")
	}

	if err := s.repo.Delete(targetURI); err != nil {
		return err
	}
	
	return s.searchRepo.DeleteOccurrence(targetURI)
}

func (s *occurrenceService) GetTaxonStats(rawID string) (*model.TaxonStats, error) {
	safeID := strings.ReplaceAll(rawID, ":", "_")
	taxonURI := "http://purl.obolibrary.org/obo/" + safeID
	return s.repo.GetTaxonStats(taxonURI, rawID)
}

func (s *occurrenceService) Search(query string, userID string) ([]repository.OccurrenceDocument, error) {
	targetTaxonID := ""

	// ★修正: 検索ワードがある場合、Fusekiに問い合わせてIDを探す
	if query != "" {
		// 例: "Carnivora" -> "ncbi:33554"
		id, err := s.repo.GetTaxonIDByLabel(query)
		
		if err == nil && id != "" {
			targetTaxonID = id
			fmt.Printf("🧠 推論ヒット: '%s' -> ID '%s' の子孫を検索します\n", query, id)
			
			// ★重要: IDが見つかったら、キーワード検索は無効化する！
			// (そうしないと「Carnivora」という文字を含まないデータがヒットしなくなる)
			query = ""
		} else {
			fmt.Printf("ℹ️ 推論ヒットなし: '%s' (通常のキーワード検索を行います)\n", query)
		}
	}

	// 検索実行
	// queryが空でも targetTaxonID があれば、祖先フィルタで検索される
	return s.searchRepo.Search(query, userID, targetTaxonID)
}
