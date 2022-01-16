// util.go contains conversion functions for the various db models to protobufs

package twirp

import (
	"strings"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
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

func convertPodFromDB(pr *db.Podcast, podCon *podcast.PodController) (*protos.Podcast, error) {
	cats, err := podCon.ConvertCategories(pr.Category)
	if err != nil {
		return nil, err
	}
	return dbPodToProto(pr, cats), nil
}

// func convertPodsFromDB(prs []db.Podcast, podCon *podcast.PodController) ([]db.Podcast, error) {
// 	var err error
// 	pods := make([]db.Podcast, len(prs))
// 	for i := range prs {
// 		pods[i], err = convertPodFromDB(&prs[i], podCon)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}
// 	return pods, nil
// }

func dbPodToProto(pr *db.Podcast, cats []podcast.Category) *protos.Podcast {
	return &protos.Podcast{
		Id:            pr.ID.String(),
		Title:         pr.Title,
		Author:        pr.Author,
		Episodic:      pr.Episodic,
		Summary:       pr.Summary,
		Link:          pr.LinkURL,
		Image:         &protos.Image{Url: pr.ImageURL},
		Explicit:      pr.Explicit,
		Language:      pr.Language,
		Keywords:      strings.Split(strings.ReplaceAll(pr.Keywords, " ", ""), ","),
		Category:      podCatsToProtoCats(cats),
		PubDate:       timestamppb.New(pr.PubDate),
		LastBuildDate: timestamppb.New(pr.PubDate),
		Rss:           pr.RSSURL,
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
