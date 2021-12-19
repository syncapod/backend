package grpc

import (
	"context"
	"log"
	"net/http"

	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AdminService struct {
	*protos.UnimplementedAdminServer
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
		return nil, status.Error(codes.Internal, "error http.Get(url): "+err.Error())
	}
	defer rssReq.Body.Close()
	pod, err := a.rssCon.AddNewPodcast(req.Url, rssReq.Body)
	if err != nil {
		return nil, status.Error(codes.Internal, "error AddNewPodcast(): "+err.Error())
	}
	log.Println("podcast cats:", pod.Category)
	protoPod, err := convertPodFromDB(pod, a.podCon)
	if err != nil {
		return nil, status.Error(codes.Internal, "error converting db pod to proto pod: "+err.Error())
	}
	return &protos.AddPodRes{Podcast: protoPod}, nil
}

func (a *AdminService) RefreshPodcast(ctx context.Context, req *protos.RefPodReq) (*protos.RefPodRes, error) {
	err := a.rssCon.UpdatePodcasts()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &protos.RefPodRes{}, nil
}
