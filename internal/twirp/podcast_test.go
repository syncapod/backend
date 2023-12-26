// Package TestMain() located in auth_test.go
package twirp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	protos "github.com/sschwartz96/syncapod-backend/internal/gen"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
	"github.com/stretchr/testify/require"
	"github.com/twitchtv/twirp"
)

var (
	testPodID, testSeshID pgtype.UUID
	testEpi1, testEpi2    db.Episode
)

func setupPodDB() error {
	// for podcast_test
	var err error
	queries := db.New(dbpg)

	testPod := db.InsertPodcastParams{Author: "Sam Schwartz", Description: "Syncapod Podcast", LinkUrl: "https://syncapod.com/podcast", ImageUrl: "http://syncapod.com/logo.png", Language: "en", Category: []int32{1, 2, 3}, Explicit: "clean", RssUrl: "https://syncapod.com/podcast.rss", PubDate: util.PGNow()}
	testPod2 := db.InsertPodcastParams{Author: "Simon Schwartz", Description: "Syncapod Podcast 2", LinkUrl: "https://syncapod.com/podcast2", ImageUrl: "http://syncapod.com/logo.png", Language: "en", Category: []int32{1, 2, 3}, Explicit: "explicit", RssUrl: "https://syncapod.com/podcast2.rss", PubDate: util.PGNow()}
	pod1, err := queries.InsertPodcast(context.Background(), testPod)
	if err != nil {
		return fmt.Errorf("failed to insert podcast1: %v", err)
	}
	testPodID = pod1.ID
	pod2, err := queries.InsertPodcast(context.Background(), testPod2)
	if err != nil {
		return fmt.Errorf("failed to insert podcast2: %v", err)
	}

	testEpi1Params := db.InsertEpisodeParams{PodcastID: pod1.ID, Title: "Test Episode", Episode: 123, PubDate: util.PGFromTime(time.Unix(1000, 0))}
	testEpi1, err = queries.InsertEpisode(context.Background(), testEpi1Params)
	if err != nil {
		return fmt.Errorf("failed to insert episode: %v", err)
	}
	testEpi2Params := db.InsertEpisodeParams{PodcastID: pod1.ID, Title: "Test Episode 2", Episode: 124, PubDate: util.PGFromTime(time.Unix(1001, 0))}
	testEpi2, err = queries.InsertEpisode(context.Background(), testEpi2Params)
	if err != nil {
		return fmt.Errorf("failed to insert episode: %v", err)
	}

	testSub := db.InsertSubscriptionParams{UserID: testUserID, PodcastID: pod1.ID, CompletedIds: []pgtype.UUID{testEpi1.ID}, InProgressIds: []pgtype.UUID{testEpi2.ID}}
	testSub2 := db.InsertSubscriptionParams{UserID: testUserID, PodcastID: pod2.ID, CompletedIds: []pgtype.UUID{}, InProgressIds: []pgtype.UUID{}}
	if err = queries.InsertSubscription(context.Background(), testSub); err != nil {
		return fmt.Errorf("failed to insert sub: %v", err)
	}
	if err = queries.InsertSubscription(context.Background(), testSub2); err != nil {
		return fmt.Errorf("failed to insert sub: %v", err)
	}

	testUserEpi := db.UpsertUserEpisodeParams{EpisodeID: testEpi1.ID, UserID: testUserID, LastSeen: util.PGFromTime(time.Now()), OffsetMillis: 123456, Played: false}
	if err = queries.UpsertUserEpisode(context.Background(), testUserEpi); err != nil {
		return fmt.Errorf("failed to insert user episode: %v", err)
	}

	// insert user session to mimic user already authenticated
	insertTestSeshParams := db.InsertSessionParams{UserID: testUserID, LoginTime: util.PGNow(), LastSeenTime: util.PGNow(), Expires: util.PGFromTime(time.Now().Add(time.Hour)), UserAgent: "testUserAgent"}
	testSesh, err := queries.InsertSession(context.Background(), db.InsertSessionParams(insertTestSeshParams))
	if err != nil {
		return fmt.Errorf("failed to insert user session: %v", err)
	}
	testSeshID = testSesh.ID
	return nil
}

func Test_PodcastGRPC(t *testing.T) {
	// add metadata for authorization
	header := make(http.Header)
	id, err := util.StringFromPGUUID(testSeshID)
	header.Set(authTokenKey, id)
	ctx, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		t.Fatalf("Twirp could not add add headers: %v", err)
	}

	client := protos.NewPodProtobufClient("http://localhost:8081", http.DefaultClient, twirp.WithClientPathPrefix("/rpc/podcast"))

	// GetPodcast
	pod, err := client.GetPodcast(ctx, &protos.GetPodReq{Id: uuid.UUID(testPodID.Bytes).String()})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	require.Equal(t, nil, err)
	require.NotNil(t, pod)

	// GetEpisodes
	epis, err := client.GetEpisodes(ctx,
		&protos.GetEpiReq{
			Id:    uuid.UUID(testPodID.Bytes).String(),
			Start: 0,
			End:   2,
		},
	)
	if err != nil {
		t.Fatalf("GetEpisodes() error: %v", err.Error())
	}
	require.Equal(t, 2, len(epis.Episodes))
	podCon, err := podcast.NewPodController(db.New(dbpg))
	if err != nil {
		t.Fatalf("Error creating podcast controller to conver episode from database to protos: %v", err)
	}
	epi2, err := podCon.ConvertEpiFromDB(&testEpi2)
	epi1, err := podCon.ConvertEpiFromDB(&testEpi1)
	require.Equal(t, epi2, epis.Episodes[0])
	require.Equal(t, epi1, epis.Episodes[1])

	// GetUserEpisode
	userEpi, err := client.GetUserEpisode(ctx,
		&protos.GetUserEpiReq{EpiID: epi1.Id})
	if err != nil {
		log.Println(err)
	}
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, userEpi)

	// UpsertUserEpisode
	userEpi.Offset = 9999
	res, err := client.UpsertUserEpisode(ctx, userEpi)
	if err != nil {
		log.Println("userepisode:", err)
	}
	require.Equal(t, nil, err)
	require.NotEqual(t, nil, res)

	// GetSubscriptions
	subs, err := client.GetSubscriptions(ctx, &protos.GetSubReq{})
	require.Equal(t, nil, err)
	require.NotEmpty(t, subs.Subscriptions)

	// GetUserLastPlayed
	lastPlayRes, err := client.GetUserLastPlayed(ctx, &protos.GetUserLastPlayedReq{})
	require.Equal(t, nil, err)
	require.NotEmpty(t, lastPlayRes.Episode)
	require.NotEmpty(t, lastPlayRes.Podcast)
	require.NotEmpty(t, lastPlayRes.Millis)
}
