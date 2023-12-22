package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PodcastStore struct {
	db *pgxpool.Pool
}

func NewPodcastStore(db *pgxpool.Pool) *PodcastStore {
	return &PodcastStore{db: db}
}

type scanner interface {
	Scan(...interface{}) error
}

// scanPodcastRows is a helper method to scan mutiple rows in podcast slice
func scanPodcastRows(rows pgx.Rows, p []Podcast) ([]Podcast, error) {
	for rows.Next() {
		temp := &Podcast{}
		if err := scanPodcastRow(rows, temp); err != nil {
			return p, fmt.Errorf("scanPodcastRows() error scanning row: %v", err)
		}
		p = append(p, *temp)
	}
	if err := rows.Err(); err != nil {
		return p, fmt.Errorf("scanPodcastRows() error while reading: %v", err)
	}
	return p, nil
}

// scanPodcastRow is a helper method to scan row into a podcast struct
func scanPodcastRow(row scanner, p *Podcast) error {
	return row.Scan(&p.ID, &p.Title, &p.Description, &p.ImageURL, &p.Language, &p.Category, &p.Explicit, &p.Author, &p.LinkURL, &p.OwnerName, &p.OwnerEmail, &p.Episodic, &p.Copyright, &p.Block, &p.Complete, &p.PubDate, &p.Keywords, &p.Summary, &p.RSSURL)
}

// scanEpisodeRows is helper method that scans mutiple rows in an episode slice
func scanEpisodeRows(rows pgx.Rows, e []Episode) ([]Episode, error) {
	for rows.Next() {
		temp := &Episode{}
		if err := scanEpisodeRow(rows, temp); err != nil {
			return e, fmt.Errorf("scanEpisodeRows() error scanning episode row: %v", err)
		}
		e = append(e, *temp)
	}
	if err := rows.Err(); err != nil {
		return e, fmt.Errorf("scanEpisodeRows() error while reading: %v", err)
	}
	return e, nil
}

// scanPodcastRow is a helper method to scan row into a podcast struct
func scanEpisodeRow(row scanner, e *Episode) error {
	return row.Scan(&e.ID, &e.Title, &e.EnclosureURL, &e.EnclosureLength, &e.EnclosureType, &e.PubDate, &e.Description, &e.Duration, &e.LinkURL, &e.ImageURL, &e.ImageTitle, &e.Explicit, &e.Episode, &e.Season, &e.EpisodeType, &e.Subtitle, &e.Summary, &e.Encoded, &e.PodcastID)
}

// Podcast stuff
func (ps *PodcastStore) InsertPodcast(ctx context.Context, p *Podcast) error {
	_, err := ps.db.Exec(ctx, "INSERT INTO Podcasts(id,title,description,image_url,language,category,explicit,author,link_url,owner_name,owner_email,episodic,copyright,block,complete,pub_date,keywords,summary,rss_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)",
		&p.ID, &p.Title, &p.Description, &p.ImageURL, &p.Language, &p.Category, &p.Explicit, &p.Author, &p.LinkURL, &p.OwnerName, &p.OwnerEmail, &p.Episodic, &p.Copyright, &p.Block, &p.Complete, &p.PubDate, &p.Keywords, &p.Summary, &p.RSSURL)
	if err != nil {
		return fmt.Errorf("InsertPodcast() error: %v", err)
	}
	return nil
}

func (ps *PodcastStore) FindPodcastByID(ctx context.Context, id uuid.UUID) (*Podcast, error) {
	p := &Podcast{}
	row := ps.db.QueryRow(ctx, "SELECT * FROM Podcasts WHERE id=$1", id)
	err := scanPodcastRow(row, p)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastByID() error: %v", err)
	}
	return p, nil
}

func (ps *PodcastStore) FindPodcastByRSS(ctx context.Context, rssURL string) (*Podcast, error) {
	p := &Podcast{}
	row := ps.db.QueryRow(ctx, "SELECT * FROM Podcasts WHERE rss_url=$1", rssURL)
	err := scanPodcastRow(row, p)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastByRSS() error: %v", err)
	}
	return p, nil
}

func (ps *PodcastStore) FindPodcastsByRange(ctx context.Context, start int, end int) ([]Podcast, error) {
	limit := end - start
	offset := start
	rows, err := ps.db.Query(ctx, "SELECT * FROM Podcasts LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastsByRange() error: %v", err)
	}
	return scanPodcastRows(rows, []Podcast{})
}

func (ps *PodcastStore) SearchPodcasts(ctx context.Context, search string) ([]Podcast, error) {
	search = strings.ReplaceAll(search, " ", "&")
	rows, err := ps.db.Query(ctx, `SELECT * FROM podcasts
								   WHERE id IN (SELECT podcast_id
									  FROM podcasts_search, to_tsquery($1) query
									  WHERE search @@ query
									  ORDER BY ts_rank(search,query)
								   );`, &search)
	if err != nil {
		return nil, fmt.Errorf("SearchPodcasts() error on query: %v", err)
	}
	return scanPodcastRows(rows, []Podcast{})
}

// Episode stuff

func (p *PodcastStore) InsertEpisode(ctx context.Context, e *Episode) error {
	_, err := p.db.Exec(ctx, `INSERT INTO Episodes(id,title,enclosure_url,enclosure_length,enclosure_type,pub_date,description,duration,link_url,image_url,image_title,explicit,episode,season,episode_type,subtitle,summary,encoded,podcast_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		&e.ID, &e.Title, &e.EnclosureURL, &e.EnclosureLength, &e.EnclosureType, &e.PubDate, &e.Description, &e.Duration, &e.LinkURL, &e.ImageURL, &e.ImageTitle, &e.Explicit, &e.Episode, &e.Season, &e.EpisodeType, &e.Subtitle, &e.Summary, &e.Encoded, &e.PodcastID)
	if err != nil {
		return fmt.Errorf("InsertEpisode() error: %v", err)
	}
	return nil
}

func (p *PodcastStore) FindEpisodeByID(ctx context.Context, epiID uuid.UUID) (*Episode, error) {
	row := p.db.QueryRow(ctx, "SELECT * FROM Episodes WHERE id=$1", &epiID)
	epi := &Episode{}
	err := scanEpisodeRow(row, epi)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeByID() error: %v", err)
	}
	return epi, nil
}

func (p *PodcastStore) FindLatestEpisode(ctx context.Context, podID uuid.UUID) (*Episode, error) {
	row := p.db.QueryRow(ctx, "SELECT * FROM Episodes WHERE podcast_id=$1 ORDER BY pub_date DESC", &podID)
	epi := &Episode{}
	err := scanEpisodeRow(row, epi)
	if err != nil {
		return nil, fmt.Errorf("FindLatestEpisode() error: %v", err)
	}
	return epi, nil
}

func (p *PodcastStore) FindEpisodeNumber(ctx context.Context, podID uuid.UUID, season, episode int) (*Episode, error) {
	row := p.db.QueryRow(ctx, "SELECT * FROM Episodes WHERE (podcast_id=$1 AND episode=$2)", &podID, &episode)
	epi := &Episode{}
	err := scanEpisodeRow(row, epi)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeNumber() error: %v", err)
	}
	return epi, nil
}

func (p *PodcastStore) FindEpisodeByURL(ctx context.Context, podID uuid.UUID, mp3URL string) (*Episode, error) {
	row := p.db.QueryRow(ctx, "SELECT * FROM Episodes WHERE (podcast_id=$1 AND enclosure_url=$2)", &podID, &mp3URL)
	epi := &Episode{}
	err := scanEpisodeRow(row, epi)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeByURL() error: %v", err)
	}
	return epi, nil
}

func (ps *PodcastStore) FindEpisodesByRange(ctx context.Context, podID uuid.UUID, start, end int64) ([]Episode, error) {
	limit := end - start
	offset := start
	rows, err := ps.db.Query(ctx,
		"SELECT * FROM Episodes WHERE podcast_id=$1 ORDER BY pub_date DESC LIMIT $2 OFFSET $3 ",
		podID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastsByRange() error: %v", err)
	}
	return scanEpisodeRows(rows, []Episode{})
}

func (p *PodcastStore) InsertCategory(ctx context.Context, cat *Category) error {
	_, err := p.db.Exec(ctx, "INSERT INTO Categories(id,name,parent_id) VALUES($1,$2,$3)",
		cat.ID, cat.Name, cat.ParentID)
	if err != nil {
		return fmt.Errorf("InsertCategory() error inserting cateogry: %v", err)
	}
	return nil
}

func (p *PodcastStore) FindAllCategories(ctx context.Context) ([]Category, error) {
	cats := []Category{}
	rows, err := p.db.Query(ctx, "SELECT * FROM Categories")
	if err != nil {
		return cats, fmt.Errorf("FindAllCategories() error: %v", err)
	}
	for rows.Next() {
		temp := Category{}
		err := rows.Scan(&temp.ID, &temp.Name, &temp.ParentID)
		if err != nil {
			return nil, fmt.Errorf("FindAllCategories() error scanning row: %v", err)
		}
		cats = append(cats, temp)
	}
	if err = rows.Err(); err != nil {
		return cats, fmt.Errorf("FindAllCategories() error reading rows: %v", err)
	}
	return cats, nil
}

func (p *PodcastStore) UpsertUserEpisode(ctx context.Context, userEpi *UserEpisode) error {
	_, err := p.db.Exec(ctx,
		`INSERT INTO UserEpisodes
		(user_id,episode_id,offset_millis,last_seen,played)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (user_id,episode_id) DO UPDATE
		SET offset_millis=$3,last_seen=$4,played=$5`,
		&userEpi.UserID, &userEpi.EpisodeID, &userEpi.OffsetMillis, &userEpi.LastSeen, &userEpi.Played,
	)
	if err != nil {
		return fmt.Errorf("UpsertUserEpisode() error: %v", err)
	}
	return nil
}

func (p *PodcastStore) FindUserEpisode(ctx context.Context, userID, epiID uuid.UUID) (*UserEpisode, error) {
	userEpi := &UserEpisode{UserID: userID, EpisodeID: epiID, LastSeen: time.Now()}
	row := p.db.QueryRow(ctx,
		"SELECT offset_millis,last_seen,played FROM UserEpisodes WHERE (user_id=$1 AND episode_id=$2)",
		&userID, &epiID)
	err := row.Scan(&userEpi.OffsetMillis, &userEpi.LastSeen, &userEpi.Played)
	if err != nil {
		return nil, fmt.Errorf("FindUserEpisode() error: %v", err)
	}
	return userEpi, nil
}

func (p *PodcastStore) FindLastUserEpi(ctx context.Context, userID uuid.UUID) (*UserEpisode, error) {
	userEpi := &UserEpisode{UserID: userID}
	row := p.db.QueryRow(ctx,
		"SELECT episode_id,offset_millis,last_seen,played FROM UserEpisodes WHERE user_id=$1 ORDER BY last_seen DESC",
		&userID)
	err := row.Scan(&userEpi.EpisodeID, &userEpi.OffsetMillis, &userEpi.LastSeen, &userEpi.Played)
	if err != nil {
		return nil, fmt.Errorf("FindLastUserEpi() error: %v", err)
	}
	return userEpi, nil
}

func (ps *PodcastStore) FindLastPlayed(ctx context.Context, userID uuid.UUID) (*UserEpisode, *Podcast, *Episode, error) {
	userEpi := &UserEpisode{UserID: userID}
	e := &Episode{}
	p := &Podcast{}
	row := ps.db.QueryRow(ctx,
		`SELECT * FROM UserEpisodes u 
		 INNER JOIN Episodes e ON u.episode_id=e.id
		 INNER JOIN Podcasts p ON e.podcast_id=p.id
		 WHERE u.user_id=$1 ORDER BY u.last_seen DESC`,
		&userID)
	err := row.Scan(&userEpi.UserID, &userEpi.EpisodeID, &userEpi.OffsetMillis, &userEpi.LastSeen, &userEpi.Played,
		&e.ID, &e.Title, &e.EnclosureURL, &e.EnclosureLength, &e.EnclosureType, &e.PubDate, &e.Description, &e.Duration, &e.LinkURL, &e.ImageURL, &e.ImageTitle, &e.Explicit, &e.Episode, &e.Season, &e.EpisodeType, &e.Subtitle, &e.Summary, &e.Encoded, &e.PodcastID,
		&p.ID, &p.Title, &p.Description, &p.ImageURL, &p.Language, &p.Category, &p.Explicit, &p.Author, &p.LinkURL, &p.OwnerName, &p.OwnerEmail, &p.Episodic, &p.Copyright, &p.Block, &p.Complete, &p.PubDate, &p.Keywords, &p.Summary, &p.RSSURL,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("FindLastUserEpi() error: %v", err)
	}
	return userEpi, p, e, nil
}

// Subscriptions

// scanSubRows is helper method that scans mutiple rows in an Subscription slice
func scanSubRows(rows pgx.Rows, s []Subscription) ([]Subscription, error) {
	for rows.Next() {
		temp := &Subscription{}
		if err := scanSubRow(rows, temp); err != nil {
			return s, fmt.Errorf("scanSubRows() error scanning subscription row: %v", err)
		}
		s = append(s, *temp)
	}
	if err := rows.Err(); err != nil {
		return s, fmt.Errorf("scanSubRows() error while reading: %v", err)
	}
	return s, nil
}

// scanPodcastRow is a helper method to scan row into a podcast struct
func scanSubRow(row scanner, s *Subscription) error {
	return row.Scan(&s.UserID, &s.PodcastID, &s.CompletedIDs, &s.InProgressIDs)
}

func (ps *PodcastStore) InsertSubscription(ctx context.Context, sub *Subscription) error {
	_, err := ps.db.Exec(ctx, "INSERT INTO Subscriptions(user_id,podcast_id,completed_ids,in_progress_ids) VALUES($1,$2,$3,$4)",
		&sub.UserID, &sub.PodcastID, &sub.CompletedIDs, &sub.InProgressIDs)
	if err != nil {
		return fmt.Errorf("InsertSubscription() error inserting subscription: %v", err)
	}
	return nil
}

func (ps *PodcastStore) DeleteSubscription(ctx context.Context, uID uuid.UUID, pID uuid.UUID) error {
	_, err := ps.db.Exec(ctx, "DELETE FROM Subscriptions WHERE user_id=$1 AND podcast_id=$2", uID, pID)
	if err != nil {
		return fmt.Errorf("DeleteSubscription() error deleting subscription from db: %v", err)
	}
	return nil
}

func (ps *PodcastStore) FindSubscriptions(ctx context.Context, userID uuid.UUID) ([]Subscription, error) {
	subs := []Subscription{}
	rows, err := ps.db.Query(ctx, "SELECT * FROM Subscriptions WHERE user_id=$1", userID)
	if err != nil {
		return nil, fmt.Errorf("FindSubscriptions() error querying db")
	}
	subs, err = scanSubRows(rows, subs)
	if err != nil {
		return nil, fmt.Errorf("FindSubscriptions() error scanning rows")
	}
	return subs, nil
}
