-- +goose Up
-- +goose StatementBegin
ALTER TABLE sources ADD COLUMN image_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sources DROP COLUMN IF EXISTS image_url;
-- +goose StatementEnd
