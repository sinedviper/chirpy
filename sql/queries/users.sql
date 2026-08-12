-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
        gen_random_uuid(), NOW(), NOW(), $1, $2

       )
    RETURNING *;

-- name: FindUserById :one
SELECT * FROM users WHERE id = $1;

-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: RemoveUsers :exec
DELETE FROM users;

-- name: UpdateUsers :exec
UPDATE users
SET email = $2, hashed_password = $3
WHERE id = $1;

-- name: UpdateChirpyUsers :exec
UPDATE users
SET is_chirpy_red = $2
WHERE id = $1;