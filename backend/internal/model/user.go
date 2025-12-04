package model

import "time"

// データベースのユーザーテーブルの形
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // JSONには含めない（隠す）
	IsSuperuser  bool      `json:is_superuser`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// フロントから送られてくる登録リクエストの形
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"` // メアド形式チェックもつける
	Password string `json:"password" binding:"required,min=8"` // 8文字以上必須
}

// ログインリクエストの形
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}
