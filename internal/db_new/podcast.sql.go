// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: podcast.sql

package db_new

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const deleteSubscription = `-- name: DeleteSubscription :exec
DELETE FROM Subscriptions 
WHERE user_id=$1 AND podcast_id=$2
`

type DeleteSubscriptionParams struct {
	UserID    pgtype.UUID
	PodcastID pgtype.UUID
}

func (q *Queries) DeleteSubscription(ctx context.Context, arg DeleteSubscriptionParams) error {
	_, err := q.db.Exec(ctx, deleteSubscription, arg.UserID, arg.PodcastID)
	return err
}

const findAllCategories = `-- name: FindAllCategories :many
SELECT id, name, parent_id FROM Categories
`

func (q *Queries) FindAllCategories(ctx context.Context) ([]Category, error) {
	rows, err := q.db.Query(ctx, findAllCategories)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Category
	for rows.Next() {
		var i Category
		if err := rows.Scan(&i.ID, &i.Name, &i.ParentID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findEpisodeByID = `-- name: FindEpisodeByID :one
SELECT id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id FROM Episodes WHERE id=$1
`

func (q *Queries) FindEpisodeByID(ctx context.Context, id pgtype.UUID) (Episode, error) {
	row := q.db.QueryRow(ctx, findEpisodeByID, id)
	var i Episode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.EnclosureUrl,
		&i.EnclosureLength,
		&i.EnclosureType,
		&i.PubDate,
		&i.Description,
		&i.Duration,
		&i.LinkUrl,
		&i.ImageUrl,
		&i.ImageTitle,
		&i.Explicit,
		&i.Episode,
		&i.Season,
		&i.EpisodeType,
		&i.Subtitle,
		&i.Summary,
		&i.Encoded,
		&i.PodcastID,
	)
	return i, err
}

const findEpisodeByURL = `-- name: FindEpisodeByURL :one
SELECT id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id FROM Episodes
WHERE (podcast_id=$1 AND enclosure_url=$2)
`

type FindEpisodeByURLParams struct {
	PodcastID    pgtype.UUID
	EnclosureUrl string
}

func (q *Queries) FindEpisodeByURL(ctx context.Context, arg FindEpisodeByURLParams) (Episode, error) {
	row := q.db.QueryRow(ctx, findEpisodeByURL, arg.PodcastID, arg.EnclosureUrl)
	var i Episode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.EnclosureUrl,
		&i.EnclosureLength,
		&i.EnclosureType,
		&i.PubDate,
		&i.Description,
		&i.Duration,
		&i.LinkUrl,
		&i.ImageUrl,
		&i.ImageTitle,
		&i.Explicit,
		&i.Episode,
		&i.Season,
		&i.EpisodeType,
		&i.Subtitle,
		&i.Summary,
		&i.Encoded,
		&i.PodcastID,
	)
	return i, err
}

const findEpisodeNumber = `-- name: FindEpisodeNumber :one
SELECT id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id FROM Episodes
WHERE (podcast_id=$1 AND episode=$2)
`

type FindEpisodeNumberParams struct {
	PodcastID pgtype.UUID
	Episode   int32
}

func (q *Queries) FindEpisodeNumber(ctx context.Context, arg FindEpisodeNumberParams) (Episode, error) {
	row := q.db.QueryRow(ctx, findEpisodeNumber, arg.PodcastID, arg.Episode)
	var i Episode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.EnclosureUrl,
		&i.EnclosureLength,
		&i.EnclosureType,
		&i.PubDate,
		&i.Description,
		&i.Duration,
		&i.LinkUrl,
		&i.ImageUrl,
		&i.ImageTitle,
		&i.Explicit,
		&i.Episode,
		&i.Season,
		&i.EpisodeType,
		&i.Subtitle,
		&i.Summary,
		&i.Encoded,
		&i.PodcastID,
	)
	return i, err
}

const findEpisodesByRange = `-- name: FindEpisodesByRange :many
SELECT id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id FROM Episodes
WHERE podcast_id=$1 
ORDER BY pub_date DESC
LIMIT $2 OFFSET $3
`

type FindEpisodesByRangeParams struct {
	PodcastID pgtype.UUID
	Limit     int64
	Offset    int64
}

func (q *Queries) FindEpisodesByRange(ctx context.Context, arg FindEpisodesByRangeParams) ([]Episode, error) {
	rows, err := q.db.Query(ctx, findEpisodesByRange, arg.PodcastID, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Episode
	for rows.Next() {
		var i Episode
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.EnclosureUrl,
			&i.EnclosureLength,
			&i.EnclosureType,
			&i.PubDate,
			&i.Description,
			&i.Duration,
			&i.LinkUrl,
			&i.ImageUrl,
			&i.ImageTitle,
			&i.Explicit,
			&i.Episode,
			&i.Season,
			&i.EpisodeType,
			&i.Subtitle,
			&i.Summary,
			&i.Encoded,
			&i.PodcastID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findLastPlayed = `-- name: FindLastPlayed :one
SELECT userepisodes.user_id, userepisodes.episode_id, userepisodes.offset_millis, userepisodes.last_seen, userepisodes.played, episodes.id, episodes.title, episodes.enclosure_url, episodes.enclosure_length, episodes.enclosure_type, episodes.pub_date, episodes.description, episodes.duration, episodes.link_url, episodes.image_url, episodes.image_title, episodes.explicit, episodes.episode, episodes.season, episodes.episode_type, episodes.subtitle, episodes.summary, episodes.encoded, episodes.podcast_id, podcasts.id, podcasts.title, podcasts.description, podcasts.image_url, podcasts.language, podcasts.category, podcasts.explicit, podcasts.author, podcasts.link_url, podcasts.owner_name, podcasts.owner_email, podcasts.episodic, podcasts.copyright, podcasts.block, podcasts.complete, podcasts.pub_date, podcasts.keywords, podcasts.summary, podcasts.rss_url
FROM UserEpisodes
INNER JOIN Episodes ON UserEpisodes.episode_id=Episodes.id
INNER JOIN Podcasts ON Episodes.podcast_id=Podcasts.id
WHERE UserEpisodes.user_id=$1 
ORDER BY UserEpisodes.last_seen DESC
LIMIT 1
`

type FindLastPlayedRow struct {
	Userepisode Userepisode
	Episode     Episode
	Podcast     Podcast
}

func (q *Queries) FindLastPlayed(ctx context.Context, userID pgtype.UUID) (FindLastPlayedRow, error) {
	row := q.db.QueryRow(ctx, findLastPlayed, userID)
	var i FindLastPlayedRow
	err := row.Scan(
		&i.Userepisode.UserID,
		&i.Userepisode.EpisodeID,
		&i.Userepisode.OffsetMillis,
		&i.Userepisode.LastSeen,
		&i.Userepisode.Played,
		&i.Episode.ID,
		&i.Episode.Title,
		&i.Episode.EnclosureUrl,
		&i.Episode.EnclosureLength,
		&i.Episode.EnclosureType,
		&i.Episode.PubDate,
		&i.Episode.Description,
		&i.Episode.Duration,
		&i.Episode.LinkUrl,
		&i.Episode.ImageUrl,
		&i.Episode.ImageTitle,
		&i.Episode.Explicit,
		&i.Episode.Episode,
		&i.Episode.Season,
		&i.Episode.EpisodeType,
		&i.Episode.Subtitle,
		&i.Episode.Summary,
		&i.Episode.Encoded,
		&i.Episode.PodcastID,
		&i.Podcast.ID,
		&i.Podcast.Title,
		&i.Podcast.Description,
		&i.Podcast.ImageUrl,
		&i.Podcast.Language,
		&i.Podcast.Category,
		&i.Podcast.Explicit,
		&i.Podcast.Author,
		&i.Podcast.LinkUrl,
		&i.Podcast.OwnerName,
		&i.Podcast.OwnerEmail,
		&i.Podcast.Episodic,
		&i.Podcast.Copyright,
		&i.Podcast.Block,
		&i.Podcast.Complete,
		&i.Podcast.PubDate,
		&i.Podcast.Keywords,
		&i.Podcast.Summary,
		&i.Podcast.RssUrl,
	)
	return i, err
}

const findLastUserEpi = `-- name: FindLastUserEpi :one
SELECT episode_id,offset_millis,last_seen,played
FROM UserEpisodes
WHERE user_id=$1
ORDER BY last_seen DESC
`

type FindLastUserEpiRow struct {
	EpisodeID    pgtype.UUID
	OffsetMillis int64
	LastSeen     pgtype.Timestamptz
	Played       bool
}

func (q *Queries) FindLastUserEpi(ctx context.Context, userID pgtype.UUID) (FindLastUserEpiRow, error) {
	row := q.db.QueryRow(ctx, findLastUserEpi, userID)
	var i FindLastUserEpiRow
	err := row.Scan(
		&i.EpisodeID,
		&i.OffsetMillis,
		&i.LastSeen,
		&i.Played,
	)
	return i, err
}

const findLatestEpisode = `-- name: FindLatestEpisode :one
SELECT id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id FROM Episodes 
WHERE podcast_id=$1 
ORDER BY pub_date DESC
`

func (q *Queries) FindLatestEpisode(ctx context.Context, podcastID pgtype.UUID) (Episode, error) {
	row := q.db.QueryRow(ctx, findLatestEpisode, podcastID)
	var i Episode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.EnclosureUrl,
		&i.EnclosureLength,
		&i.EnclosureType,
		&i.PubDate,
		&i.Description,
		&i.Duration,
		&i.LinkUrl,
		&i.ImageUrl,
		&i.ImageTitle,
		&i.Explicit,
		&i.Episode,
		&i.Season,
		&i.EpisodeType,
		&i.Subtitle,
		&i.Summary,
		&i.Encoded,
		&i.PodcastID,
	)
	return i, err
}

const findPodcastByID = `-- name: FindPodcastByID :one
SELECT id, title, description, image_url, language, category, explicit, author, link_url, owner_name, owner_email, episodic, copyright, block, complete, pub_date, keywords, summary, rss_url FROM Podcasts
WHERE id=$1
`

func (q *Queries) FindPodcastByID(ctx context.Context, id pgtype.UUID) (Podcast, error) {
	row := q.db.QueryRow(ctx, findPodcastByID, id)
	var i Podcast
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Description,
		&i.ImageUrl,
		&i.Language,
		&i.Category,
		&i.Explicit,
		&i.Author,
		&i.LinkUrl,
		&i.OwnerName,
		&i.OwnerEmail,
		&i.Episodic,
		&i.Copyright,
		&i.Block,
		&i.Complete,
		&i.PubDate,
		&i.Keywords,
		&i.Summary,
		&i.RssUrl,
	)
	return i, err
}

const findPodcastByRSS = `-- name: FindPodcastByRSS :one
SELECT id, title, description, image_url, language, category, explicit, author, link_url, owner_name, owner_email, episodic, copyright, block, complete, pub_date, keywords, summary, rss_url FROM Podcasts
WHERE rss_url=$1
`

func (q *Queries) FindPodcastByRSS(ctx context.Context, rssUrl string) (Podcast, error) {
	row := q.db.QueryRow(ctx, findPodcastByRSS, rssUrl)
	var i Podcast
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Description,
		&i.ImageUrl,
		&i.Language,
		&i.Category,
		&i.Explicit,
		&i.Author,
		&i.LinkUrl,
		&i.OwnerName,
		&i.OwnerEmail,
		&i.Episodic,
		&i.Copyright,
		&i.Block,
		&i.Complete,
		&i.PubDate,
		&i.Keywords,
		&i.Summary,
		&i.RssUrl,
	)
	return i, err
}

const findPodcastsByRange = `-- name: FindPodcastsByRange :many
SELECT id, title, description, image_url, language, category, explicit, author, link_url, owner_name, owner_email, episodic, copyright, block, complete, pub_date, keywords, summary, rss_url FROM Podcasts 
LIMIT $1 OFFSET $2
`

type FindPodcastsByRangeParams struct {
	Limit  int64
	Offset int64
}

func (q *Queries) FindPodcastsByRange(ctx context.Context, arg FindPodcastsByRangeParams) ([]Podcast, error) {
	rows, err := q.db.Query(ctx, findPodcastsByRange, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Podcast
	for rows.Next() {
		var i Podcast
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Description,
			&i.ImageUrl,
			&i.Language,
			&i.Category,
			&i.Explicit,
			&i.Author,
			&i.LinkUrl,
			&i.OwnerName,
			&i.OwnerEmail,
			&i.Episodic,
			&i.Copyright,
			&i.Block,
			&i.Complete,
			&i.PubDate,
			&i.Keywords,
			&i.Summary,
			&i.RssUrl,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findSubscriptions = `-- name: FindSubscriptions :many
SELECT user_id, podcast_id, completed_ids, in_progress_ids FROM Subscriptions
WHERE user_id=$1
`

func (q *Queries) FindSubscriptions(ctx context.Context, userID pgtype.UUID) ([]Subscription, error) {
	rows, err := q.db.Query(ctx, findSubscriptions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Subscription
	for rows.Next() {
		var i Subscription
		if err := rows.Scan(
			&i.UserID,
			&i.PodcastID,
			&i.CompletedIds,
			&i.InProgressIds,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const findUserEpisode = `-- name: FindUserEpisode :one
SELECT user_id, episode_id, offset_millis, last_seen, played
FROM UserEpisodes
WHERE (user_id=$1 AND episode_id=$2)
`

type FindUserEpisodeParams struct {
	UserID    pgtype.UUID
	EpisodeID pgtype.UUID
}

func (q *Queries) FindUserEpisode(ctx context.Context, arg FindUserEpisodeParams) (Userepisode, error) {
	row := q.db.QueryRow(ctx, findUserEpisode, arg.UserID, arg.EpisodeID)
	var i Userepisode
	err := row.Scan(
		&i.UserID,
		&i.EpisodeID,
		&i.OffsetMillis,
		&i.LastSeen,
		&i.Played,
	)
	return i, err
}

const insertCategory = `-- name: InsertCategory :exec
INSERT INTO Categories(id,name,parent_id)
VALUES($1,$2,$3)
`

type InsertCategoryParams struct {
	ID       int32
	Name     string
	ParentID int32
}

func (q *Queries) InsertCategory(ctx context.Context, arg InsertCategoryParams) error {
	_, err := q.db.Exec(ctx, insertCategory, arg.ID, arg.Name, arg.ParentID)
	return err
}

const insertEpisode = `-- name: InsertEpisode :one
INSERT INTO Episodes(title,enclosure_url,enclosure_length,enclosure_type,pub_date,description,duration,link_url,image_url,image_title,explicit,episode,season,episode_type,subtitle,summary,encoded,podcast_id)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING id, title, enclosure_url, enclosure_length, enclosure_type, pub_date, description, duration, link_url, image_url, image_title, explicit, episode, season, episode_type, subtitle, summary, encoded, podcast_id
`

type InsertEpisodeParams struct {
	Title           string
	EnclosureUrl    string
	EnclosureLength int64
	EnclosureType   string
	PubDate         pgtype.Timestamptz
	Description     string
	Duration        int64
	LinkUrl         string
	ImageUrl        string
	ImageTitle      string
	Explicit        string
	Episode         int32
	Season          int32
	EpisodeType     string
	Subtitle        string
	Summary         string
	Encoded         string
	PodcastID       pgtype.UUID
}

func (q *Queries) InsertEpisode(ctx context.Context, arg InsertEpisodeParams) (Episode, error) {
	row := q.db.QueryRow(ctx, insertEpisode,
		arg.Title,
		arg.EnclosureUrl,
		arg.EnclosureLength,
		arg.EnclosureType,
		arg.PubDate,
		arg.Description,
		arg.Duration,
		arg.LinkUrl,
		arg.ImageUrl,
		arg.ImageTitle,
		arg.Explicit,
		arg.Episode,
		arg.Season,
		arg.EpisodeType,
		arg.Subtitle,
		arg.Summary,
		arg.Encoded,
		arg.PodcastID,
	)
	var i Episode
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.EnclosureUrl,
		&i.EnclosureLength,
		&i.EnclosureType,
		&i.PubDate,
		&i.Description,
		&i.Duration,
		&i.LinkUrl,
		&i.ImageUrl,
		&i.ImageTitle,
		&i.Explicit,
		&i.Episode,
		&i.Season,
		&i.EpisodeType,
		&i.Subtitle,
		&i.Summary,
		&i.Encoded,
		&i.PodcastID,
	)
	return i, err
}

const insertPodcast = `-- name: InsertPodcast :one
INSERT INTO Podcasts(title,description,image_url,language,category,explicit,author,link_url,owner_name,owner_email,episodic,copyright,block,complete,pub_date,keywords,summary,rss_url)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING id, title, description, image_url, language, category, explicit, author, link_url, owner_name, owner_email, episodic, copyright, block, complete, pub_date, keywords, summary, rss_url
`

type InsertPodcastParams struct {
	Title       string
	Description string
	ImageUrl    string
	Language    string
	Category    []int32
	Explicit    string
	Author      string
	LinkUrl     string
	OwnerName   string
	OwnerEmail  string
	Episodic    pgtype.Bool
	Copyright   string
	Block       pgtype.Bool
	Complete    pgtype.Bool
	PubDate     pgtype.Timestamptz
	Keywords    string
	Summary     string
	RssUrl      string
}

func (q *Queries) InsertPodcast(ctx context.Context, arg InsertPodcastParams) (Podcast, error) {
	row := q.db.QueryRow(ctx, insertPodcast,
		arg.Title,
		arg.Description,
		arg.ImageUrl,
		arg.Language,
		arg.Category,
		arg.Explicit,
		arg.Author,
		arg.LinkUrl,
		arg.OwnerName,
		arg.OwnerEmail,
		arg.Episodic,
		arg.Copyright,
		arg.Block,
		arg.Complete,
		arg.PubDate,
		arg.Keywords,
		arg.Summary,
		arg.RssUrl,
	)
	var i Podcast
	err := row.Scan(
		&i.ID,
		&i.Title,
		&i.Description,
		&i.ImageUrl,
		&i.Language,
		&i.Category,
		&i.Explicit,
		&i.Author,
		&i.LinkUrl,
		&i.OwnerName,
		&i.OwnerEmail,
		&i.Episodic,
		&i.Copyright,
		&i.Block,
		&i.Complete,
		&i.PubDate,
		&i.Keywords,
		&i.Summary,
		&i.RssUrl,
	)
	return i, err
}

const insertSubscription = `-- name: InsertSubscription :exec
INSERT INTO Subscriptions(user_id,podcast_id,completed_ids,in_progress_ids)
VALUES($1,$2,$3,$4)
`

type InsertSubscriptionParams struct {
	UserID        pgtype.UUID
	PodcastID     pgtype.UUID
	CompletedIds  []pgtype.UUID
	InProgressIds []pgtype.UUID
}

func (q *Queries) InsertSubscription(ctx context.Context, arg InsertSubscriptionParams) error {
	_, err := q.db.Exec(ctx, insertSubscription,
		arg.UserID,
		arg.PodcastID,
		arg.CompletedIds,
		arg.InProgressIds,
	)
	return err
}

const searchPodcasts = `-- name: SearchPodcasts :many
SELECT id, title, description, image_url, language, category, explicit, author, link_url, owner_name, owner_email, episodic, copyright, block, complete, pub_date, keywords, summary, rss_url FROM podcasts
WHERE id IN (SELECT podcast_id
	FROM podcasts_search, to_tsquery($1) query
	WHERE search @@ query
	ORDER BY ts_rank(search,query)
)
`

func (q *Queries) SearchPodcasts(ctx context.Context, toTsquery string) ([]Podcast, error) {
	rows, err := q.db.Query(ctx, searchPodcasts, toTsquery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Podcast
	for rows.Next() {
		var i Podcast
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Description,
			&i.ImageUrl,
			&i.Language,
			&i.Category,
			&i.Explicit,
			&i.Author,
			&i.LinkUrl,
			&i.OwnerName,
			&i.OwnerEmail,
			&i.Episodic,
			&i.Copyright,
			&i.Block,
			&i.Complete,
			&i.PubDate,
			&i.Keywords,
			&i.Summary,
			&i.RssUrl,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const upsertUserEpisode = `-- name: UpsertUserEpisode :exec
INSERT INTO UserEpisodes(user_id,episode_id,offset_millis,last_seen,played)
VALUES($1,$2,$3,$4,$5)
ON CONFLICT (user_id,episode_id) DO UPDATE
SET offset_millis=$3,last_seen=$4,played=$5
`

type UpsertUserEpisodeParams struct {
	UserID       pgtype.UUID
	EpisodeID    pgtype.UUID
	OffsetMillis int64
	LastSeen     pgtype.Timestamptz
	Played       bool
}

func (q *Queries) UpsertUserEpisode(ctx context.Context, arg UpsertUserEpisodeParams) error {
	_, err := q.db.Exec(ctx, upsertUserEpisode,
		arg.UserID,
		arg.EpisodeID,
		arg.OffsetMillis,
		arg.LastSeen,
		arg.Played,
	)
	return err
}
