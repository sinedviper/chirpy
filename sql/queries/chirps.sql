-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
           gen_random_uuid(), NOW(), NOW(), $1, $2

       )
    RETURNING *;

-- name: GetChirps :many
SELECT * FROM chirps ORDER BY
  CASE WHEN sqlc.arg(sort_direction)::text = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort_direction)::text = 'desc' THEN created_at END DESC;

-- name: GetChirp :one
SELECT * FROM chirps WHERE id = $1;

-- name: DeleteChirpById :exec
DELETE FROM chirps WHERE id = $1;

-- name: GetChirpsByAuthor :many
SELECT * FROM chirps WHERE chirps.user_id = $1 ORDER BY
  CASE WHEN sqlc.arg(sort_direction)::text = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort_direction)::text = 'desc' THEN created_at END DESC;