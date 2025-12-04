-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


UPDATE users 
SET is_superuser = TRUE 
WHERE email = 'info@ecology-bio.net';


-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
