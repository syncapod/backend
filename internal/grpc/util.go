// util.go contains conversion functions for the various db models to protobufs

package grpc

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUserEpiFromDB(u *db.UserEpisode) *protos.UserEpisode {
	return &protos.UserEpisode{
		UserID:    u.UserID.String(),
		EpisodeID: u.UserID.String(),
		Offset:    u.OffsetMillis,
		LastSeen:  timestamppb.New(u.LastSeen),
		Played:    u.Played,
	}
}

func convertUserFromDB(ur *db.UserRow) *protos.User {
	return &protos.User{
		Id:       ur.ID.String(),
		Email:    ur.Email,
		Username: ur.Username,
		DOB:      timestamppb.New(ur.Birthdate),
	}
}

func convertPodFromDB(pr *db.Podcast, cats []podcast.Category) *protos.Podcast {
	return &protos.Podcast{
		Id:            pr.ID.String(),
		Title:         pr.Title,
		Summary:       pr.Summary,
		Author:        pr.Author,
		Category:      podCatsToProtoCats(cats),
		Explicit:      pr.Explicit,
		Image:         &protos.Image{Url: pr.ImageURL},
		Keywords:      strings.Split(strings.ReplaceAll(pr.Keywords, " ", ""), ","),
		Language:      pr.Language,
		LastBuildDate: timestamppb.New(pr.PubDate), // TODO: proper build date?
		Link:          pr.LinkURL,
		PubDate:       timestamppb.New(pr.PubDate),
		Rss:           pr.RSSURL,
		Episodic:      pr.Episodic,
	}
}

func convertEpiFromDB(er *db.Episode) *protos.Episode {
	return &protos.Episode{
		Id:             er.ID.String(),
		PodcastID:      er.PodcastID.String(),
		Title:          er.Title,
		Subtitle:       er.Subtitle,
		EpisodeType:    er.EpisodeType,
		Image:          &protos.Image{Title: er.ImageTitle, Url: er.ImageURL},
		PubDate:        timestamppb.New(er.PubDate),
		Description:    er.Description,
		Summary:        er.Summary,
		Season:         int32(er.Season),
		Episode:        int32(er.Episode),
		Explicit:       er.Explicit,
		MP3URL:         er.EnclosureURL,
		DurationMillis: er.Duration,
		Encoded:        er.Encoded,
	}
}

func convertPodsFromDB(podCon *podcast.PodController, p []db.Podcast) ([]*protos.Podcast, error) {
	protoPods := make([]*protos.Podcast, len(p))
	for i := range p {
		cats, err := podCon.ConvertCategories(p[i].Category)
		if err != nil {
			return nil, fmt.Errorf("Could not convert podcast categories: %v", err)
		}
		protoPods[i] = convertPodFromDB(&p[i], cats)
	}
	return protoPods, nil
}

func convertEpisFromDB(e []db.Episode) []*protos.Episode {
	protoEpis := make([]*protos.Episode, len(e))
	for i := range e {
		protoEpis[i] = convertEpiFromDB(&e[i])
	}
	return protoEpis
}

func podCatsToProtoCats(podCats []podcast.Category) []*protos.Category {
	protoCats := []*protos.Category{}
	for i := range podCats {
		protoCats = append(protoCats, podCatToProtoCat(podCats[i]))
	}
	return protoCats
}

func podCatToProtoCat(podCat podcast.Category) *protos.Category {
	return &protos.Category{
		Category: podCatsToProtoCats(podCat.Subcategories),
		Text:     podCat.Name,
	}
}

func convertSubFromDB(s []db.Subscription) []*protos.Subscription {
	subs := []*protos.Subscription{}
	for i := range s {
		subs = append(subs, &protos.Subscription{
			UserID:        s[i].UserID.String(),
			PodcastID:     s[i].PodcastID.String(),
			CompletedIDs:  convertUUIDsToStrings(s[i].CompletedIDs),
			InProgressIDs: convertUUIDsToStrings(s[i].InProgressIDs),
		})
	}
	return subs
}

func convertUUIDsToStrings(u []uuid.UUID) []string {
	s := []string{}
	for i := range u {
		s = append(s, u[i].String())
	}
	return s
}
