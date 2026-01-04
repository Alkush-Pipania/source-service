-- name: GetSourceByID :one
SELECT id, user_id, collection_id, type, status, title, original_url, s3_bucket, s3_key, content_hash, created_at, image_url
FROM sources
WHERE id = $1;

-- name: UpdateSourceStatus :exec
UPDATE sources 
SET status = $2
WHERE id = $1;

-- name: UpdateSourceTitleAndImage :exec
UPDATE sources 
SET title = $2, image_url = $3
WHERE id = $1;