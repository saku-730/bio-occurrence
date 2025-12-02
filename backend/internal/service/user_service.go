package service

import (
	"github.com/saku-730/bio-occurrence/backend/internal/model"
	"github.com/saku-730/bio-occurrence/backend/internal/repository"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(req model.RegisterRequest) (*model.User, error)
	Login(req model.LoginRequest) (*model.User, error)
}

type authService struct {
	userRepo repository.UserRepository
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{userRepo: userRepo}
}

func (s *authService) Register(req model.RegisterRequest) (*model.User, error) {
	// 1. 重複チェック
	// (DBのUNIQUE制約でも弾けるけど、親切なエラーメッセージのためにここでもチェックするのが一般的)
	existingUser, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, fmt.Errorf("このメールアドレスは既に使用されている")
	}

	// 2. パスワードのハッシュ化
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing failed: %w", err)
	}

	// 3. ユーザーモデルの作成
	newUser := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPass),
	}

	// 4. 保存
	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *authService) Login(req model.LoginRequest) (*model.User, error) {
	// 1. ユーザーを探す
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("ユーザーが見つからないか、パスワードが違います")
	}

	// 2. パスワード照合 (Hash vs Raw)
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("ユーザーが見つからないか、パスワードが違います")
	}

	// 3. 成功したらユーザー情報を返す
	return user, nil
}
