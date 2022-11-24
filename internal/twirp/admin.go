package twirp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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
	urlStr := strings.TrimSpace(req.Url)
	if urlStr == "" {
		return nil, twirp.InvalidArgumentError("Url", "Url cannot be empty")
	}

	url, err := url.Parse(urlStr)
	if err != nil {
		return nil, twirp.InvalidArgumentError("Url", fmt.Sprintf("Url could not be parse: %s", err.Error()))
	}

	pod, err := a.rssCon.AddPodcast(ctx, url)
	if err != nil {
		return nil, twirp.InternalErrorf("Could not add podcast: %w", err)
	}

	protoPod, err := convertPodFromDB(pod, a.podCon)
	if err != nil {
		return nil, twirp.InternalErrorf("Could not convert podcast to protobuf object: %w", err)
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
