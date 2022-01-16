package twirp

import (
	"context"
	"log"
	"net/http"

	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/twitchtv/twirp"
)

type AdminService struct {
	podCon *podcast.PodController
	rssCon *podcast.RSSController
}

func NewAdminService(podCon *podcast.PodController, rssCon *podcast.RSSController) *AdminService {
	return &AdminService{
		podCon: podCon,
		rssCon: rssCon,
	}
}

// Podcasts
func (a *AdminService) AddPodcast(ctx context.Context, req *protos.AddPodReq) (*protos.AddPodRes, error) {
	rssReq, err := http.Get(req.Url)
	if err != nil {
		return nil, twirp.Internal.Errorf("error http.Get(url): %w", err)
	}
	defer rssReq.Body.Close()
	pod, err := a.rssCon.AddNewPodcast(req.Url, rssReq.Body)
	if err != nil {
		return nil, twirp.Internal.Errorf("error AddNewPodcast(): %w", err)
	}
	log.Println("podcast cats:", pod.Category)
	protoPod, err := convertPodFromDB(pod, a.podCon)
	if err != nil {
		return nil, twirp.Internal.Errorf("error converting db pod to proto pod: %w", err)
	}
	return &protos.AddPodRes{Podcast: protoPod}, nil
}

func (a *AdminService) RefreshPodcast(ctx context.Context, req *protos.RefPodReq) (*protos.RefPodRes, error) {
	err := a.rssCon.UpdatePodcasts()
	if err != nil {
		return nil, twirp.Internal.Errorf("%v", err)
	}
	return &protos.RefPodRes{}, nil
}

func (a *AdminService) SearchPodcasts(ctx context.Context, req *protos.SearchPodReq) (*protos.SearchPodRes, error) {
	dbPods, err := a.podCon.SearchPodcasts(ctx, req.Text)
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	pods, err := convertPodsFromDB(a.podCon, dbPods)
	if err != nil {
		return nil, twirp.InternalError(err.Error())
	}
	return &protos.SearchPodRes{
		Podcasts: pods,
	}, nil
}
