package twirp

import (
	"context"
	"fmt"
	"log"

	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
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
	pid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, twirp.InvalidArgument.Errorf("Could not parse podcast id: %w", err)
	}
	dbPod, err := p.podCon.FindPodcastByID(ctx, pid)
	if err != nil {
		return nil, twirp.NotFound.Errorf("Could not find podcast error: %w", err)
	}
	pod, err := convertPodFromDB(dbPod, p.podCon)
	if err != nil {
		return nil, twirp.Internal.Errorf("Error converting podcast model: %w", err)
	}
	return pod, nil
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
	dbEpis, err := p.podCon.FindEpisodesByRange(ctx, podID, req.Start, req.End)
	if err != nil {
		return nil, twirp.Internal.Errorf("Could not find episodes by range: %w", err)
	}
	epis := convertEpisFromDB(dbEpis)
	return &protos.Episodes{Episodes: epis}, nil
}

// GetUserEpisode returns the user playback metadata via episode id & user id
func (p *PodcastService) GetUserEpisode(ctx context.Context, req *protos.GetUserEpiReq) (*protos.UserEpisode, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, twirp.Unauthenticated.Error("User is not authenticated")
	}
	epiID, err := uuid.Parse(req.EpiID)
	if err != nil {
		return nil, twirp.InvalidArgument.Errorf("Could not parse episode uuid: %w", err)
	}
	dbUserEpi, err := p.podCon.FindUserEpisode(ctx, userID, epiID)
	if err != nil {
		return nil, twirp.Internal.Errorf("Could not retrieve user episode: %w", err)
	}
	return convertUserEpiFromDB(dbUserEpi), nil
}

// UpsertUserEpisode updates the user playback metadata via episode id & user id
func (p *PodcastService) UpsertUserEpisode(ctx context.Context, userEpiReq *protos.UserEpisode) (*protos.Response, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, twirp.Unauthenticated.Error("User is not authenticated")
	}
	epiID, err := uuid.Parse(userEpiReq.EpisodeID)
	if err != nil {
		return nil, twirp.InvalidArgument.Error("Could not parse episode UUID")
	}
	if userEpiReq.LastSeen == nil {
		userEpiReq.LastSeen = ptypes.TimestampNow()
	}
	userEpi := &db.UserEpisode{
		UserID:       userID,
		EpisodeID:    epiID,
		OffsetMillis: userEpiReq.Offset,
		LastSeen:     userEpiReq.LastSeen.AsTime(),
		Played:       userEpiReq.Played,
	}
	err = p.podCon.UpsertUserEpisode(ctx, userEpi)
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
		log.Println("GetSubscriptions() error getting subs:", err)
		return &protos.Subscriptions{}, nil
	}

	return &protos.Subscriptions{Subscriptions: convertSubFromDB(subs)}, nil
}

// GetUserLastPlayed returns the last episode the user was playing & metadata
func (p *PodcastService) GetUserLastPlayed(ctx context.Context, req *protos.GetUserLastPlayedReq) (*protos.LastPlayedRes, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("GetUserLastPlayed() error getting user id: %v", err)
	}

	userEpi, pod, epi, err := p.podCon.FindLastPlayed(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserLastPlayed() error: %v", err)
	}
	protoPod, err := convertPodFromDB(pod, p.podCon)
	if err != nil {
		return nil, twirp.InternalErrorf("Could not convert podcast model from DB: %w", err)
	}
	return &protos.LastPlayedRes{
		Podcast: protoPod,
		Episode: convertEpiFromDB(epi),
		Millis:  userEpi.OffsetMillis,
	}, nil
}

func getUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	userData, ok := ctx.Value(twirpHeaderKey{}).(twirpCtxData)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("getUserIDFromContext() error could not extract data from context")
	}

	return userData.user.ID, nil
}
