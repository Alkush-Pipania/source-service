-- name: CreateSourceContent :exec
INSERT INTO source_contents (source_id, content_text)
VALUES ($1, $2);

-- name: GetSourceContentBySourceID :many
SELECT * FROM source_contents WHERE source_id = $1;