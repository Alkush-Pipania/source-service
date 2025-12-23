-- name: UpdateSourceStatus :exec
UPDATE sources 
SET status = $2
WHERE id = $1;