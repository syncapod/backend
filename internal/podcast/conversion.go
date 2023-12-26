package podcast

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (c *PodController) convertUserEpiFromDB(u *db.Userepisode) (*protos.UserEpisode, error) {
	userID, err := util.StringFromPGUUID(u.UserID)
	if err != nil {
		return nil, fmt.Errorf("convertUserEpiFromDB error converting user id: %w", err)
	}
	episodeID, err := util.StringFromPGUUID(u.EpisodeID)
	if err != nil {
		return nil, fmt.Errorf("convertUserEpiFromDB error converting episode id: %w", err)
	}
	return &protos.UserEpisode{
		UserID:    userID,
		EpisodeID: episodeID,
		Offset:    u.OffsetMillis,
		LastSeen:  timestamppb.New(u.LastSeen.Time),
		Played:    u.Played,
	}, nil
}

func (c *PodController) ConvertPodFromDB(pr *db.Podcast) (*protos.Podcast, error) {
	cats, err := c.ConvertCategories(pr.Category)
	if err != nil {
		return nil, err
	}
	return c.dbPodToProto(pr, cats)
}

func (c *PodController) dbPodToProto(pr *db.Podcast, cats []Category) (*protos.Podcast, error) {
	if !pr.ID.Valid {
		return nil, errors.New("error podcast id is not valid")
	}
	return &protos.Podcast{
		Id:            uuid.UUID(pr.ID.Bytes).String(),
		Title:         pr.Title,
		Summary:       pr.Summary,
		Author:        pr.Author,
		Category:      c.podCatsToProtoCats(cats),
		Explicit:      pr.Explicit,
		Image:         &protos.Image{Url: pr.ImageUrl},
		Keywords:      strings.Split(strings.ReplaceAll(pr.Keywords, " ", ""), ","),
		Language:      pr.Language,
		LastBuildDate: timestamppb.New(pr.PubDate.Time), // TODO: proper build date?
		Link:          pr.LinkUrl,
		PubDate:       timestamppb.New(pr.PubDate.Time),
		Rss:           pr.RssUrl,
		Episodic:      pr.Episodic.Bool,
	}, nil
}

func (c *PodController) ConvertEpiFromDB(er *db.Episode) (*protos.Episode, error) {
	if !er.ID.Valid {
		return nil, errors.New("error episode id is not valid")
	}
	if !er.PodcastID.Valid {
		return nil, errors.New("error episode's podcast id is not valid")
	}
	return &protos.Episode{
		Id:             uuid.UUID(er.ID.Bytes).String(),
		PodcastID:      uuid.UUID(er.PodcastID.Bytes).String(),
		Title:          er.Title,
		Subtitle:       er.Subtitle,
		EpisodeType:    er.EpisodeType,
		Image:          &protos.Image{Title: er.ImageTitle, Url: er.ImageUrl},
		PubDate:        timestamppb.New(er.PubDate.Time),
		Description:    er.Description,
		Summary:        er.Summary,
		Season:         int32(er.Season),
		Episode:        int32(er.Episode),
		Explicit:       er.Explicit,
		MP3URL:         er.EnclosureUrl,
		DurationMillis: er.Duration,
		Encoded:        er.Encoded,
	}, nil
}

func (c *PodController) convertPodsFromDB(p []db.Podcast) ([]*protos.Podcast, error) {
	var err error
	protoPods := make([]*protos.Podcast, len(p))
	for i := range p {
		protoPods[i], err = c.ConvertPodFromDB(&p[i])
		if err != nil {
			return nil, err
		}
	}
	return protoPods, nil
}

func (c *PodController) convertEpisFromDB(e []db.Episode) ([]*protos.Episode, error) {
	var err error
	protoEpis := make([]*protos.Episode, len(e))
	for i := range e {
		protoEpis[i], err = c.ConvertEpiFromDB(&e[i])
		if err != nil {
			return nil, err
		}
	}
	return protoEpis, nil
}

func (c *PodController) podCatsToProtoCats(podCats []Category) []*protos.Category {
	protoCats := []*protos.Category{}
	for i := range podCats {
		protoCats = append(protoCats, c.podCatToProtoCat(podCats[i]))
	}
	return protoCats
}

func (c *PodController) podCatToProtoCat(podCat Category) *protos.Category {
	return &protos.Category{
		Category: c.podCatsToProtoCats(podCat.Subcategories),
		Text:     podCat.Name,
	}
}

func (c *PodController) convertSubFromDB(s []db.Subscription) ([]*protos.Subscription, error) {

	subs := []*protos.Subscription{}
	for i := range s {
		userID, err := util.StringFromPGUUID(s[i].UserID)
		if err != nil {
			return nil, fmt.Errorf("convertSubFromDB error converting user id: %w", err)
		}
		podcastID, err := util.StringFromPGUUID(s[i].PodcastID)
		if err != nil {
			return nil, fmt.Errorf("convertSubFromDB error converting podcast id: %w", err)
		}
		completedIDs, err := convertPGUUIDsToStrings(s[i].CompletedIds)
		if err != nil {
			return nil, fmt.Errorf("convertSubFromDB error converting completedIDs: %w", err)
		}
		inProgressIDs, err := convertPGUUIDsToStrings(s[i].InProgressIds)
		if err != nil {
			return nil, fmt.Errorf("convertSubFromDB error converting inProgressIDs: %w", err)
		}
		subs = append(subs, &protos.Subscription{
			UserID:        userID,
			PodcastID:     podcastID,
			CompletedIDs:  completedIDs,
			InProgressIDs: inProgressIDs,
		})
	}
	return subs, nil
}

func convertPGUUIDsToStrings(u []pgtype.UUID) ([]string, error) {
	s := []string{}
	for i := range u {
		sUUID, err := util.StringFromPGUUID(u[i])
		if err != nil {
			return nil, err
		}
		s = append(s, sUUID)
	}
	return s, nil
}

func convertUUIDsToStrings(u []uuid.UUID) []string {
	s := []string{}
	for i := range u {
		s = append(s, u[i].String())
	}
	return s
}
