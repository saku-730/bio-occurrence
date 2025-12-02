-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- groupsテーブルから owner_id カラムを削除する
ALTER TABLE groups DROP COLUMN owner_id;


-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back

-- もし戻す場合は、owner_id カラムを復活させる
-- (元々 NOT NULL だったけど、既存データがあるとエラーになるので NULL許可 で戻すのが無難)
ALTER TABLE groups ADD COLUMN owner_id UUID REFERENCES users(id) ON DELETE CASCADE;
