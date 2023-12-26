package twirp

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/twitchtv/twirp"
)

// PodcastService is the gRPC service for podcast
type PodcastService struct {
	podCon *podcast.PodController
}

// NewPodcastService creates a new *PodcastService
func NewPodcastService(podCon *podcast.PodController) *PodcastService {
	return &PodcastService{podCon: podCon}
}

// GetPodcast returns a podcast via id
func (p *PodcastService) GetPodcast(ctx context.Context, req *protos.GetPodReq) (*protos.Podcast, error) {
	podcast, err := p.podCon.FindPodcastByID(ctx, req.Id)
	if err != nil {
		return nil, twirp.NotFound.Errorf("Could not find podcast error: %w", err)
	}
	return podcast, nil
}

// GetEpisodes returns a list of episodes via podcast id
func (p *PodcastService) GetEpisodes(ctx context.Context, req *protos.GetEpiReq) (*protos.Episodes, error) {
	podID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, twirp.InvalidArgument.Error("Could not parse podcast UUID")
	}
	if req.End == 0 {
		req.End = 10
	}
	epis, err := p.podCon.FindEpisodesByRange(ctx, util.PGUUID(podID), req.Start, req.End)
	if err != nil {
		return nil, twirp.Internal.Errorf("Could not find episodes by range: %w", err)
	}
	return &protos.Episodes{Episodes: epis}, nil
}

// GetUserEpisode returns the user playback metadata via episode id & user id
func (p *PodcastService) GetUserEpisode(ctx context.Context, req *protos.GetUserEpiReq) (*protos.UserEpisode, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, twirp.Unauthenticated.Error("User is not authenticated")
	}
	userEpi, err := p.podCon.FindUserEpisode(ctx, userID.String(), req.EpiID)
	if err != nil {
		return nil, twirp.Internal.Errorf("Could not retrieve user episode: %w", err)
	}
	return userEpi, nil
}

// UpsertUserEpisode updates the user playback metadata via episode id & user id
func (p *PodcastService) UpsertUserEpisode(ctx context.Context, userEpiReq *protos.UserEpisode) (*protos.Response, error) {
	err := p.podCon.UpsertUserEpisode(ctx, userEpiReq)
	if err != nil {
		return nil, twirp.Internal.Errorf("Error upserting UserEpisode: %w", err)
	}
	return &protos.Response{Success: true, Message: ""}, nil
}

// GetSubscriptions returns a list of podcasts via user id
func (p *PodcastService) GetSubscriptions(ctx context.Context, req *protos.GetSubReq) (*protos.Subscriptions, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetSubscriptions() error getting user id: %v", err)
	}
	subs, err := p.podCon.FindSubscriptions(ctx, userID)
	if err != nil {
		// TODO: 2023-12-18 understand and handle this error
		return &protos.Subscriptions{}, nil
	}

	return subs, nil
}

// GetUserLastPlayed returns the last episode the user was playing & metadata
func (p *PodcastService) GetUserLastPlayed(ctx context.Context, req *protos.GetUserLastPlayedReq) (*protos.LastPlayedRes, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetUserLastPlayed() error getting user id: %v", err)
	}
	return p.podCon.FindLastPlayed(ctx, userID.String())
}

func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userData, ok := ctx.Value(twirpHeaderKey{}).(twirpCtxData)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("getUserIDFromContext() error could not extract data from context")
	}

	return userData.user.ID.Bytes, nil
}
