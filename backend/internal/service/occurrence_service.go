package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"fmt"
	"strings"
	"log"

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
	userRepo   repository.UserRepository // ★追加: ユーザー情報を引くため
}

// コンストラクタに userRepo を追加
func NewOccurrenceService(
	repo repository.OccurrenceRepository,
	searchRepo repository.SearchRepository,
	userRepo repository.UserRepository, // ★追加
) OccurrenceService {
	return &occurrenceService{
		repo:       repo,
		searchRepo: searchRepo,
		userRepo:   userRepo,
	}
}

func (s *occurrenceService) Register(userID string, req model.OccurrenceRequest) (string, error) {
	// 1. ユーザー情報を取得（検索インデックスに名前を入れるため）
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	ancestors, err := s.repo.GetAncestorIDs(req.TaxonID)
	if err != nil {
		log.Printf("⚠️ 祖先の取得に失敗: %v", err)
		ancestors = []string{req.TaxonID} // 最低限自分自身は入れる
	}

	occUUID := uuid.New().String()
	occURI := "http://my-db.org/occ/" + occUUID
	
	// 2. Fusekiに保存
	err = s.repo.Create(occURI, userID, req)
	if err != nil {
		return "", err
	}

	// 3. Meilisearchにも保存 (ユーザーIDと名前も渡す！)
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

	// N+1問題になるけど、今はシンプルにループでユーザー名を取得するのだ
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

	// ユーザー名を取得
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

func (s *occurrenceService) Search(query string, userID string) ([]repository.OccurrenceDocument, error) {
	targetTaxonID := ""

	// クエリが空でない場合、それが「分類名」かどうかを確認する
	if query != "" {
        // GetTaxonIDByName みたいなメソッドをRepoに追加して呼ぶのが綺麗だけど、
        // 既存の GetDescendantIDs のロジックを少し変えて「IDだけ返す」メソッドを作ると良い
        
        // 今回は簡易的に、Repoに新メソッドを追加せず、既存の GetDescendantIDs を流用するハックでいくなら:
        // しかし、GetDescendantIDsは「子孫全部」を返すので重い。
        // 「名前からID1つだけ」を返すメソッドをRepoに追加するのが正解なのだ。
        
        // ★Repositoryに GetTaxonIDByLabel を追加したと仮定する
		id, err := s.repo.GetTaxonIDByLabel(query)
		if err == nil && id != "" {
			targetTaxonID = id
            // IDが見つかったら、クエリ文字列は空にしてフィルタリング検索に切り替える
            // (そうしないと "Vertebrata" という文字を持たないデータがヒットしないため)
            query = ""
            fmt.Printf("🧠 推論検索: %s -> %s の子孫を検索します\n", query, targetTaxonID)
		}
	}

	// 検索実行
	return s.searchRepo.Search(query, userID, targetTaxonID)
}


func (s *occurrenceService) Modify(userID string, id string, req model.OccurrenceRequest) error {
	targetURI := "http://my-db.org/occ/" + id

	// ★追加: まず既存データを取得して所有者チェック！
	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}
	// 持ち主じゃなかったらエラー！
	if existing.OwnerID != userID {
		return fmt.Errorf("permission denied: あなたのデータではない")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return fmt.Errorf("failed to find user")
	}

	if err := s.repo.Update(targetURI, userID, req); err != nil {
		return err
	}

	ancestors, err := s.repo.GetAncestorIDs(req.TaxonID)
	if err != nil {
		// エラーなら自分自身だけ入れる
		ancestors = []string{req.TaxonID}
	}

	if err := s.repo.Update(targetURI, userID, req); err != nil {
		return err
	}
	
	// 引数を追加
	return s.searchRepo.IndexOccurrence(req, targetURI, user.ID, user.Username, ancestors)
}

func (s *occurrenceService) Remove(userID string, id string) error {
	targetURI := "http://my-db.org/occ/" + id
	
	// ★追加: 所有者チェック！
	existing, err := s.repo.FindByID(targetURI)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("not found")
	}
	if existing.OwnerID != userID {
		return fmt.Errorf("permission denied: 他人のデータは消せない")
	}

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

