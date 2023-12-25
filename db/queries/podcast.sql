-- name: InsertPodcast :one
INSERT INTO Podcasts(title,description,image_url,language,category,explicit,author,link_url,owner_name,owner_email,episodic,copyright,block,complete,pub_date,keywords,summary,rss_url)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING *;

-- name: FindPodcastByID :one
SELECT * FROM Podcasts
WHERE id=$1;

-- name: FindPodcastByRSS :one
SELECT * FROM Podcasts
WHERE rss_url=$1;

-- name: FindPodcastsByRange :many
SELECT * FROM Podcasts 
LIMIT $1 OFFSET $2;

-- name: SearchPodcasts :many
SELECT * FROM podcasts
WHERE id IN (SELECT podcast_id
	FROM podcasts_search, to_tsquery($1) query
	WHERE search @@ query
	ORDER BY ts_rank(search,query)
);

-- name: InsertEpisode :one
INSERT INTO Episodes(title,enclosure_url,enclosure_length,enclosure_type,pub_date,description,duration,link_url,image_url,image_title,explicit,episode,season,episode_type,subtitle,summary,encoded,podcast_id)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING *;

-- name: FindEpisodeByID :one
SELECT * FROM Episodes WHERE id=$1;

-- name: FindLatestEpisode :one
SELECT * FROM Episodes 
WHERE podcast_id=$1 
ORDER BY pub_date DESC;

-- name: FindEpisodeNumber :one
SELECT * FROM Episodes
WHERE (podcast_id=$1 AND episode=$2);

-- name: FindEpisodeByURL :one
SELECT * FROM Episodes
WHERE (podcast_id=$1 AND enclosure_url=$2);

-- name: FindEpisodesByRange :many
SELECT * FROM Episodes
WHERE podcast_id=$1 
ORDER BY pub_date DESC
LIMIT $2 OFFSET $3;

-- name: InsertCategory :exec
INSERT INTO Categories(id,name,parent_id)
VALUES($1,$2,$3);

-- name: FindAllCategories :many
SELECT * FROM Categories;

-- name: UpsertUserEpisode :exec
INSERT INTO UserEpisodes(user_id,episode_id,offset_millis,last_seen,played)
VALUES($1,$2,$3,$4,$5)
ON CONFLICT (user_id,episode_id) DO UPDATE
SET offset_millis=$3,last_seen=$4,played=$5;

-- name: FindUserEpisode :one
SELECT *
FROM UserEpisodes
WHERE (user_id=$1 AND episode_id=$2);

-- name: FindLastUserEpi :one
SELECT episode_id,offset_millis,last_seen,played
FROM UserEpisodes
WHERE user_id=$1
ORDER BY last_seen DESC;

-- name: FindLastPlayed :one
SELECT * FROM UserEpisodes
INNER JOIN Episodes ON UserEpisodes.episode_id=Episodes.id
INNER JOIN Podcasts ON Episodes.podcast_id=Podcasts.id
WHERE UserEpisodes.user_id=$1 
ORDER BY UserEpisodes.last_seen DESC;

-- name: InsertSubscription :exec
INSERT INTO Subscriptions(user_id,podcast_id,completed_ids,in_progress_ids)
VALUES($1,$2,$3,$4);

-- name: DeleteSubscription :exec
DELETE FROM Subscriptions 
WHERE user_id=$1 AND podcast_id=$2;

-- name: FindSubscriptions :many
SELECT * FROM Subscriptions
WHERE user_id=$1;
