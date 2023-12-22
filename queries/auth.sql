-- name: InsertUser :exec
INSERT INTO Users (
	id,email,username,birthdate,password_hash, created, last_seen
) VALUES (
	$1,$2,$3,$4,$5,$6,$7
);
