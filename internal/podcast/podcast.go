package podcast

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
)

type PodController struct {
	*db.PodcastStore
	catCache *CategoryCache
}

func NewPodController(podStore *db.PodcastStore) (*PodController, error) {
	cats, err := podStore.FindAllCategories(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NewPodController() error creating CategoryCache: %v", err)
	}
	catCache := newCategoryCache(cats, podStore)
	return &PodController{podStore, catCache}, nil
}

func (p *PodController) ConvertCategories(ids []int) ([]Category, error) {
	return p.catCache.LookupIDs(ids)
}

func (c *PodController) DoesPodcastExist(ctx context.Context, rssURL string) bool {
	_, err := c.FindPodcastByRSS(ctx, rssURL)
	return err == nil
}

func (c *PodController) DoesEpisodeExist(ctx context.Context, podID uuid.UUID, mp3URL string) bool {
	_, err := c.FindEpisodeByURL(ctx, podID, mp3URL)
	return err == nil
}
