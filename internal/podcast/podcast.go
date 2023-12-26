package podcast

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

type PodController struct {
	queries  *db.Queries
	catCache *CategoryCache
}

func NewPodController(queries *db.Queries) (*PodController, error) {
	cats, err := queries.FindAllCategories(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NewPodController() error creating CategoryCache: %v", err)
	}
	catCache := newCategoryCache(cats, queries)
	return &PodController{queries: queries, catCache: catCache}, nil
}

func (p *PodController) ConvertCategories(ids []int32) ([]Category, error) {
	return p.catCache.LookupIDs(ids)
}

func (c *PodController) DoesPodcastExist(ctx context.Context, rssURL string) bool {
	_, err := c.queries.FindPodcastByRSS(ctx, rssURL)
	// TODO: better error handling to make sure the error was actually "no rows"
	return err == nil
}

func (c *PodController) DoesEpisodeExist(ctx context.Context, podID pgtype.UUID, mp3URL string) bool {
	_, err := c.queries.FindEpisodeByURL(
		ctx,
		db.FindEpisodeByURLParams{
			PodcastID:    podID,
			EnclosureUrl: mp3URL,
		})
	return err == nil
}

// Methods similar to DB queries, but act as a conversation layer

func (c *PodController) FindPodcastByID(ctx context.Context, id string) (*protos.Podcast, error) {
	podcastPGUUID, err := util.PGUUIDFromString(id)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastByID error converting id: %w", err)
	}
	dbPodcast, err := c.queries.FindPodcastByID(ctx, podcastPGUUID)
	if err != nil {
		return nil, fmt.Errorf("FindPodcastByID db error: %w", err)
	}
	return c.ConvertPodFromDB(&dbPodcast)
}

func (c *PodController) FindEpisodesByRange(ctx context.Context, podID pgtype.UUID, start, end int64) ([]*protos.Episode, error) {
	dbEpisodes, err := c.queries.FindEpisodesByRange(ctx,
		db.FindEpisodesByRangeParams{
			PodcastID: podID,
			Limit:     end - start,
			Offset:    start,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("FindEpisodesByRange db error: %w", err)
	}

	return c.convertEpisFromDB(dbEpisodes)
}

func (c *PodController) FindUserEpisode(ctx context.Context, userID, epiID string) (*protos.UserEpisode, error) {
	userPGUUID, err := util.PGUUIDFromString(userID)
	if err != nil {
		return nil, fmt.Errorf("FindUserEpisode error parsing user uuid: %w", err)
	}
	episodePGUUID, err := util.PGUUIDFromString(epiID)
	if err != nil {
		return nil, fmt.Errorf("FindUserEpisode error parsing episode uuid: %w", err)
	}
	dbUserEpisode, err := c.queries.FindUserEpisode(ctx, db.FindUserEpisodeParams{
		UserID:    userPGUUID,
		EpisodeID: episodePGUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("FindUserEpisode db error: %w", err)
	}
	return c.convertUserEpiFromDB(&dbUserEpisode)
}

func (c *PodController) SearchPodcasts(ctx context.Context, searchTerm string) ([]*protos.Podcast, error) {
	dbPods, err := c.queries.SearchPodcasts(ctx, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("SearchPodcasts db error: %w", err)
	}
	return c.convertPodsFromDB(dbPods)
}

func (c *PodController) UpsertUserEpisode(ctx context.Context, userEpi *protos.UserEpisode) error {
	userID, err := util.PGUUIDFromString(userEpi.UserID)
	if err != nil {
		return fmt.Errorf("UpsertUserEpisode error parsing user uuid: %w", err)
	}
	episodeID, err := util.PGUUIDFromString(userEpi.EpisodeID)
	if err != nil {
		return fmt.Errorf("UpsertUserEpisode error parsing episode uuid: %w", err)
	}

	err = c.queries.UpsertUserEpisode(ctx, db.UpsertUserEpisodeParams{
		UserID:       userID,
		EpisodeID:    episodeID,
		OffsetMillis: userEpi.Offset,
		LastSeen:     util.PGFromTime(userEpi.LastSeen.AsTime()),
		Played:       userEpi.Played,
	})
	if err != nil {
		return fmt.Errorf("UpsertUserEpisode error on db upsert: %w", err)
	}
	return nil
}

func (c *PodController) FindSubscriptions(ctx context.Context, userID uuid.UUID) (*protos.Subscriptions, error) {
	dbSubs, err := c.queries.FindSubscriptions(ctx, util.PGUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("FindSubscriptions error on db query: %w", err)
	}

	subs, err := c.convertSubFromDB(dbSubs)
	if err != nil {
		return nil, fmt.Errorf("FindSubscriptions error on subscription model conversion: %w", err)
	}

	return &protos.Subscriptions{Subscriptions: subs}, nil
}

func (c *PodController) FindLastPlayed(ctx context.Context, userID string) (*protos.LastPlayedRes, error) {
	userPGUUID, err := util.PGUUIDFromString(userID)
	if err != nil {
		return nil, fmt.Errorf("FindLastPlayed error converting id: %w", err)
	}
	lastPlayedRow, err := c.queries.FindLastPlayed(ctx, userPGUUID)
	if err != nil {
		return nil, fmt.Errorf("FindLastPlayed error on db query: %w", err)
	}
	podcast, err := c.ConvertPodFromDB(&lastPlayedRow.Podcast)
	if err != nil {
		return nil, fmt.Errorf("FindLastPlayed error on converting podcast: %w", err)
	}
	episode, err := c.ConvertEpiFromDB(&lastPlayedRow.Episode)
	if err != nil {
		return nil, fmt.Errorf("FindLastPlayed error on converting episode: %w", err)
	}
	return &protos.LastPlayedRes{
		Podcast: podcast,
		Episode: episode,
		Millis:  lastPlayedRow.Userepisode.OffsetMillis,
	}, nil
}

func (c *PodController) FindEpisodeNumber(ctx context.Context, podcastID string, episodeNumber int) (*protos.Episode, error) {
	podPGUUUID, err := util.PGUUIDFromString(podcastID)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeNumber error on converting podcast id: %w", err)
	}
	dbEpisode, err := c.queries.FindEpisodeNumber(ctx, db.FindEpisodeNumberParams{
		PodcastID: podPGUUUID,
		Episode:   int32(episodeNumber),
	})
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeNumber error on db query: %w", err)
	}
	episode, err := c.ConvertEpiFromDB(&dbEpisode)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeNumber error on converting episode: %w", err)
	}

	return episode, nil
}

func (c *PodController) FindLatestEpisode(ctx context.Context, podcastID string) (*protos.Episode, error) {
	podPGUUUID, err := util.PGUUIDFromString(podcastID)
	if err != nil {
		return nil, fmt.Errorf("FindLatestEpisode error on converting podcast id: %w", err)
	}
	dbEpisode, err := c.queries.FindLatestEpisode(ctx, podPGUUUID)
	if err != nil {
		return nil, fmt.Errorf("FindLatestEpisode error on db query: %w", err)
	}
	episode, err := c.ConvertEpiFromDB(&dbEpisode)
	if err != nil {
		return nil, fmt.Errorf("FindLatestEpisode error on converting episode: %w", err)
	}
	return episode, nil
}

func (c *PodController) FindEpisodeByID(ctx context.Context, id string) (*protos.Episode, error) {
	epiPGID, err := util.PGUUIDFromString(id)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeByID error on converting episode id: %w", err)
	}
	dbEpisode, err := c.queries.FindEpisodeByID(ctx, epiPGID)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeByID error on db query: %w", err)
	}
	episode, err := c.ConvertEpiFromDB(&dbEpisode)
	if err != nil {
		return nil, fmt.Errorf("FindEpisodeByID error on converting episode: %w", err)
	}
	return episode, nil
}

// func (c *PodController)
// func (c *PodController)
// func (c *PodController)
// func (c *PodController)
