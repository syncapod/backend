package podcast

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

type PodController struct {
	queries  *db_new.Queries
	catCache *CategoryCache
}

func NewPodController(queries *db_new.Queries) (*PodController, error) {
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
		db_new.FindEpisodeByURLParams{
			PodcastID:    podID,
			EnclosureUrl: mp3URL,
		})
	return err == nil
}

// Proxy methods

// TODO: return twirp models instead of db models

func (c *PodController) FindPodcastByID(ctx context.Context, id pgtype.UUID) (db_new.Podcast, error) {
	return c.queries.FindPodcastByID(ctx, id)
}

func (c *PodController) FindEpisodesByRange(ctx context.Context, podID pgtype.UUID, start, end int64) ([]db_new.Episode, error) {
	return c.queries.FindEpisodesByRange(ctx,
		db_new.FindEpisodesByRangeParams{
			PodcastID: podID,
			Limit:     end - start,
			Offset:    start,
		},
	)
}

func (c *PodController) FindUserEpisode(ctx context.Context, userID, epiID uuid.UUID) (db_new.Userepisode, error) {
	return c.queries.FindUserEpisode(ctx, db_new.FindUserEpisodeParams{
		UserID:    util.PGUUID(userID),
		EpisodeID: util.PGUUID(epiID),
	})
}
