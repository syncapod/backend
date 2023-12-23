---- User ----

-- name: InsertUser :exec
INSERT INTO Users (
	id,email,username,birthdate,password_hash, created, last_seen
) VALUES (
	$1,$2,$3,$4,$5,$6,$7
);

-- name: GetUserByID :one
SELECT * FROM Users WHERE id=$1;

-- name: GetUserByEmail :one
SELECT * FROM Users WHERE LOWER(email)=LOWER($1);

-- name: GetUserByUsername :one
SELECT * FROM Users WHERE LOWER(username)=LOWER($1);

-- name: UpdateUser :exec
UPDATE Users SET email=$2,username=$3,birthdate=$4,last_seen=$5 WHERE id=$1;

-- name: UpdateUserPassword :exec
UPDATE Users SET password_hash=$1 WHERE id=$2;

-- name: DeleteUser :exec
DELETE FROM Users WHERE id=$1;

-- name: InsertSession :one
INSERT INTO Sessions (id, user_id, login_time, last_seen_time, expires, user_agent) 
VALUES($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetSession :one
SELECT * FROM Sessions WHERE id=$1;

-- name: UpdateSession :exec
UPDATE Sessions SET user_id=$2,login_time=$3,last_seen_time=$4,expires=$5,user_agent=$6 WHERE id=$1;

-- name: DeleteSession :exec
DELETE FROM Sessions WHERE id=$1;

-- name: GetSessionAndUser :one
SELECT sqlc.embed(Sessions), sqlc.embed(Users)
FROM Sessions 
JOIN Users ON Sessions.user_id = Users.id 
WHERE Sessions.id = $1;


-- name: InsertAuthCode :exec
INSERT INTO AuthCodes (code,client_id,user_id,scope,expires)
VALUES($1,$2,$3,$4,$5);

-- name: GetAuthCode :one
SELECT * FROM AuthCodes WHERE code=$1;

-- name: DeleteAuthCode :exec
DELETE FROM AuthCodes WHERE code=$1;

-- name: InsertAccessToken :exec
INSERT INTO AccessTokens (token,auth_code,refresh_token,user_id,created,expires)
VALUES($1,$2,$3,$4,$5,$6);

-- name: GetAccessTokenByRefresh :one
SELECT * FROM AccessTokens WHERE refresh_token=$1;

-- name: DeleteAccessToken :exec
DELETE FROM AccessTokens WHERE token=$1;

-- name: GetAccessTokenAndUser :one
SELECT * FROM AccessTokens a
JOIN Users u ON a.user_id=u.id
WHERE a.token=$1;


