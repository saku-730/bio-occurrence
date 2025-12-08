package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"fmt"
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
	Search(query string, taxonQuery string, currentUserID string) ([]repository.OccurrenceDocument, error)
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

	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	
	// 3. Fusekiに保存
	err = s.repo.Create(occURI, userID, req)
	if err != nil {
		return "", err
	}

	// 4. Meilisearchにも保存
	if err := s.searchRepo.IndexOccurrence(req, occURI, user.ID, user.Username); err != nil {
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

	// 1. 既存データのチェック (所有権確認)
	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}
	
	// 操作ユーザー情報の取得 (権限チェックと更新用)
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user")
	}

	// 所有者でもスーパーユーザーでもなければエラー
	if existing.OwnerID != userID && !user.IsSuperuser {
		return fmt.Errorf("permission denied: あなたのデータではないのだ")
	}

	// 3. Fuseki更新
	if err := s.repo.Update(targetURI, userID, req); err != nil {
		return err
	}
	
	// 4. Meilisearch更新 (ここで user 変数が必要だったのだ！)
	return s.searchRepo.IndexOccurrence(req, targetURI, user.ID, user.Username)
}

func (s *occurrenceService) Remove(userID string, id string) error {
	targetURI := "http://my-db.org/occ/" + id
	
	// 所有権チェック
	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user")
	}

	if existing.OwnerID != userID && !user.IsSuperuser {
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

func (s *occurrenceService) Search(query string, taxonQuery string, userID string) ([]repository.OccurrenceDocument, error) {
	var targetTaxonIDs []string

	if taxonQuery != "" {
		// GetDescendantIDs は、「そのTaxonおよび子孫」かつ「実際にデータが存在するID」を返してくれる
		// これにより、データがないIDまで検索クエリに含める無駄を省けるのだ
		ids, err := s.repo.GetDescendantIDs(taxonQuery)
		if err == nil && len(ids) > 0 {
			targetTaxonIDs = ids
			fmt.Printf("🧠 推論検索: '%s' の子孫を含む %d 件のIDで検索します\n", taxonQuery, len(ids))
		} else {
			fmt.Printf("⚠️ 分類名 '%s' に該当するデータ（子孫含む）が見つからなかったのだ\n", taxonQuery)
			// ヒットなしにするためにダミーを入れるか、空配列のままにして全件検索にならないように制御する
			// ここでは「空配列＝ヒットなし」として扱うため、明示的にありえない値をセットする手もあるが、
			// SearchRepo側で len > 0 のときだけフィルタ追加しているので、
			// フィルタを追加しないと「全件検索」になってしまう恐れがある。
			// なので、見つからなかった場合は「存在しないID」でフィルタして0件にするのが安全なのだ。
			targetTaxonIDs = []string{"NO_HIT"} 
		}
	}

	return s.searchRepo.Search(query, userID, targetTaxonIDs)
}
