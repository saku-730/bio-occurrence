-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- UUID生成機能を使えるようにする
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. ユーザーテーブル
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), -- UUIDを自動生成
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,        -- メアドは重複禁止
    password_hash VARCHAR(255) NOT NULL,       -- ハッシュ化されたパスワード
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. 役割テーブル (固定値: admin, member など)
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

-- 初期データの投入 (役割)
INSERT INTO roles (name) VALUES ('admin'), ('member'), ('viewer');

-- 3. グループテーブル
CREATE TABLE groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- 作成者
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. グループメンバー (中間テーブル)
CREATE TABLE group_members (
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INT REFERENCES roles(id) DEFAULT 2, -- デフォルトは member
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (group_id, user_id) -- 同じユーザーが同じグループに二重登録されないように
);

-- 5. 招待状テーブル (招待機能用におまけ)
CREATE TABLE group_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- 招待した人
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending, accepted, rejected
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS group_invitations;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "pgcrypto";
