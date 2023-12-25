// util.go contains conversion functions for the various db models to protobufs

package twirp

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/db_new"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertUserEpiFromDB(u *db_new.Userepisode) *protos.UserEpisode {
	return &protos.UserEpisode{
		UserID:    u.UserID.String(),
		EpisodeID: u.UserID.String(),
		Offset:    u.OffsetMillis,
		LastSeen:  timestamppb.New(u.LastSeen),
		Played:    u.Played,
	}
}

func convertUserFromDB(ur *db_new.User) *protos.User {
	id, _ := util.StringFromPGUUID(ur.ID)
	return &protos.User{
		Id:       id,
		Email:    ur.Email,
		Username: ur.Username,
		DOB:      timestamppb.New(ur.Birthdate.Time),
	}
}

func convertPodFromDB(pr *db_new.Podcast, podCon *podcast.PodController) (*protos.Podcast, error) {
	cats, err := podCon.ConvertCategories(pr.Category)
	if err != nil {
		return nil, err
	}
	return dbPodToProto(pr, cats)
}

func dbPodToProto(pr *db_new.Podcast, cats []podcast.Category) (*protos.Podcast, error) {
	id, err := uuid.ParseBytes(pr.ID.Bytes[:])
	if err != nil {
		return nil, err
	}
	return &protos.Podcast{
		Id:            id.String(),
		Title:         pr.Title,
		Summary:       pr.Summary,
		Author:        pr.Author,
		Category:      podCatsToProtoCats(cats),
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

func convertEpiFromDB(er *db_new.Episode) (*protos.Episode, error) {
	id, err := uuid.ParseBytes(er.ID.Bytes[:])
	if err != nil {
		return nil, err
	}
	podID, err := uuid.ParseBytes(er.PodcastID.Bytes[:])
	if err != nil {
		return nil, err
	}
	return &protos.Episode{
		Id:             id.String(),
		PodcastID:      podID.String(),
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

func convertPodsFromDB(podCon *podcast.PodController, p []db_new.Podcast) ([]*protos.Podcast, error) {
	var err error
	protoPods := make([]*protos.Podcast, len(p))
	for i := range p {
		protoPods[i], err = convertPodFromDB(&p[i], podCon)
		if err != nil {
			return nil, err
		}
	}
	return protoPods, nil
}

func convertEpisFromDB(e []db_new.Episode) ([]*protos.Episode, error) {
	var err error
	protoEpis := make([]*protos.Episode, len(e))
	for i := range e {
		protoEpis[i], err = convertEpiFromDB(&e[i])
		if err != nil {
			return nil, err
		}
	}
	return protoEpis, nil
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
