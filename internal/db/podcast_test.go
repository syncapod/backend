package db

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var (
	testPod     = &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "Syncapod Podcast", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast.rss"}
	testEpi     = &Episode{ID: uuid.New(), PodcastID: testPod.ID, Title: "Test Episode", Episode: 123}
	testUser    = &UserRow{ID: uuid.New(), Username: "dbTestUser", PasswordHash: []byte("shouldbehash")}
	testUserEpi = &UserEpisode{EpisodeID: testEpi.ID, UserID: testUser.ID, LastSeen: time.Now(), OffsetMillis: 123456, Played: false}
)

func setupPodcastDB() {
	podStore := NewPodcastStore(testDB)
	authStore := NewAuthStorePG(testDB)
	err := podStore.InsertPodcast(context.Background(), testPod)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
	err = podStore.InsertEpisode(context.Background(), testEpi)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
	err = authStore.InsertUser(context.Background(), testUser)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
	err = podStore.UpsertUserEpisode(context.Background(), testUserEpi)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
}

func Test_FindPodcast(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	pod, err := podStore.FindPodcastByID(context.Background(), testPod.ID)
	if err != nil {
		t.Fatalf("Test_FindPodcast() error: %v", err)
	}
	require.Equal(t, testPod.ID, pod.ID)
}

func Test_FindPodcastByRSS(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	pod, err := podStore.FindPodcastByRSS(context.Background(), testPod.RSSURL)
	if err != nil {
		t.Fatalf("Test_FindPodcastRSS() error: %v", err)
	}
	require.Equal(t, testPod.ID, pod.ID)
}

func Test_FindPodcastsByRange(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	pods, err := podStore.FindPodcastsByRange(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("Test_FindPodcastRSS() error: %v", err)
	}
	require.NotEmpty(t, pods)
}

func Test_InsertPodcast(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	pod := &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "Test Insert Podcast", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast_test.rss"}
	err := podStore.InsertPodcast(context.Background(), pod)
	if err != nil {
		t.Fatalf("Test_InsertPodcast() error: %v", err)
	}
}

func Test_FindLatestEpisode(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	epi, err := podStore.FindLatestEpisode(context.Background(), testPod.ID)
	if err != nil {
		t.Fatalf("Test_FindLatestEpisode() error: %v", err)
	}
	require.Equal(t, testEpi.ID, epi.ID)
}

func Test_FindEpisodeByID(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	epi, err := podStore.FindEpisodeByID(context.Background(), testEpi.ID)
	if err != nil {
		t.Fatalf("Test_FindEpisodeByID() error: %v", err)
	}
	require.Equal(t, testEpi.ID, epi.ID)
}

func Test_FindEpisodeByURL(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	epi, err := podStore.FindEpisodeByURL(context.Background(), testPod.ID, testEpi.EnclosureURL)
	if err != nil {
		t.Fatalf("Test_FindEpisodeByURL() error: %v", err)
	}
	require.Equal(t, testEpi.ID, epi.ID)
}

func Test_FindEpisodeNumber(t *testing.T) {
	podStore := NewPodcastStore(testDB)

	epiFound, err := podStore.FindEpisodeNumber(context.Background(), testEpi.PodcastID, 0, 123)
	if err != nil {
		t.Fatalf("Test_FindEpisodeNumber() error finding episode: %v", err)
	}
	require.NotNil(t, *epiFound)
}

func Test_SearchPodcasts(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	pod := &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "PostgreSQL Search Test", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast.rss"}
	err := podStore.InsertPodcast(context.Background(), pod)
	if err != nil {
		t.Fatalf("Test_SearchPodcasts() error inserting podcast: %v", err)
	}
	pods, err := podStore.SearchPodcasts(context.Background(), "search test")
	if err != nil {
		t.Fatalf("Test_SearchPodcasts() error searching for podcasts: %v", err)
	}
	if len(pods) == 0 {
		t.Fatal("Test_SearchPodcasts() error no podcasts found")
	}
	require.Equal(t, pods[0].ID, pod.ID)
}

func Test_FindAllCategories(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	cats, err := podStore.FindAllCategories(context.Background())
	if err != nil {
		t.Fatalf("Test_FindEpisodeByURL() error: %v", err)
	}
	require.NotEmpty(t, cats)
}

func Test_FindUserEpisode(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	userEpi, err := podStore.FindUserEpisode(context.Background(), testUser.ID, testEpi.ID)
	if err != nil {
		t.Fatalf("Test_FindUserEpisode() error: %v", err)
	}
	require.NotNil(t, userEpi)
}

func Test_FindLastUserEpi(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	userEpi, err := podStore.FindLastUserEpi(context.Background(), testUser.ID)
	if err != nil {
		t.Fatalf("Test_FindLastUserEpi() error: %v", err)
	}
	require.NotNil(t, userEpi)
}

func Test_FindLastUserPlayed(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	userEpi, pod, epi, err := podStore.FindLastPlayed(context.Background(), testUser.ID)
	if err != nil {
		t.Fatalf("Test_FindLastUserEpiWithEpisode() error: %v", err)
	}
	require.NotEmpty(t, userEpi)
	require.NotEmpty(t, epi)
	require.NotEmpty(t, pod)
}

func Test_UpsertUserEpisode(t *testing.T) {
	podStore := NewPodcastStore(testDB)
	upsertUserEpi := *testUserEpi
	upsertUserEpi.OffsetMillis = 654321
	err := podStore.UpsertUserEpisode(context.Background(), &upsertUserEpi)
	if err != nil {
		t.Fatalf("Test_UpsertUserEpisode() error: %v", err)
	}
	upsertUserEpi2, err := podStore.FindUserEpisode(context.Background(), upsertUserEpi.UserID, upsertUserEpi.EpisodeID)
	if err != nil {
		t.Fatalf("Test_UpsertUserEpisode() error finding user epi: %v", err)
	}
	require.Equal(t, upsertUserEpi.OffsetMillis, upsertUserEpi2.OffsetMillis)
}
