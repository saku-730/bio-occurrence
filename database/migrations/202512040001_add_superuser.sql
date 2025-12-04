-- +goose Up
-- 既存のユーザーテーブルにスーパーユーザーフラグを追加（デフォルトは false）
ALTER TABLE users ADD COLUMN is_superuser BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN is_superuser;
