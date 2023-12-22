package db

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var (
	testPod     = &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "Syncapod Podcast", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast.rss"}
	testPod2    = &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "Syncapod Podcast", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast.rss"}
	testEpi     = &Episode{ID: uuid.New(), PodcastID: testPod.ID, Title: "Test Episode", Episode: 123, PubDate: time.Unix(1000, 0)}
	testEpi2    = &Episode{ID: uuid.New(), PodcastID: testPod.ID, Title: "Test Episode 2", Episode: 124, PubDate: time.Unix(1001, 0)}
	testUser    = &UserRow{ID: uuid.New(), Username: "dbTestUser", PasswordHash: []byte("shouldbehash")}
	testUserEpi = &UserEpisode{EpisodeID: testEpi.ID, UserID: testUser.ID, LastSeen: time.Now(), OffsetMillis: 123456, Played: false}
	testSub     = &Subscription{UserID: testUser.ID, PodcastID: testPod.ID, CompletedIDs: []uuid.UUID{testEpi.ID}, InProgressIDs: []uuid.UUID{testEpi2.ID}}
	testSub2    = &Subscription{UserID: testUser.ID, PodcastID: testPod2.ID, CompletedIDs: []uuid.UUID{}, InProgressIDs: []uuid.UUID{}}
)

func setupPodcastDB() {
	podStore := NewPodcastStore(dbpg)
	authStore := NewAuthStorePG(dbpg)
	insertPodcastOrFail(podStore, testPod)
	insertPodcastOrFail(podStore, testPod2)
	insertEpisodeOrFail(podStore, testEpi)
	insertEpisodeOrFail(podStore, testEpi2)
	err := authStore.InsertUser(context.Background(), testUser)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
	err = podStore.UpsertUserEpisode(context.Background(), testUserEpi)
	if err != nil {
		log.Fatalf("db.setupPodcastDB() error: %v", err)
	}
	insertSubOrFail(podStore, testSub)
	insertSubOrFail(podStore, testSub2)
}

func insertPodcastOrFail(podStore *PodcastStore, p *Podcast) {
	err := podStore.InsertPodcast(context.Background(), p)
	if err != nil {
		log.Fatalf("db.insertPodcastOrFail() error: %v", err)
	}
}

func insertEpisodeOrFail(podStore *PodcastStore, e *Episode) {
	err := podStore.InsertEpisode(context.Background(), e)
	if err != nil {
		log.Fatalf("db.insertEpisodeOrFail() error: %v", err)
	}
}

func insertSubOrFail(podStore *PodcastStore, s *Subscription) {
	log.Println("inserting sub:", s)
	err := podStore.InsertSubscription(context.Background(), s)
	if err != nil {
		time.Sleep(time.Minute)
		log.Fatalf("db.insertSubOrFail() error: %v", err)
	}
}

func Test_FindPodcast(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	pod, err := podStore.FindPodcastByID(context.Background(), testPod.ID)
	if err != nil {
		t.Fatalf("Test_FindPodcast() error: %v", err)
	}
	require.Equal(t, testPod.ID, pod.ID)
}

func Test_FindPodcastByRSS(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	pod, err := podStore.FindPodcastByRSS(context.Background(), testPod.RSSURL)
	if err != nil {
		t.Fatalf("Test_FindPodcastRSS() error: %v", err)
	}
	require.Equal(t, testPod.ID, pod.ID)
}

func Test_FindPodcastsByRange(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	pods, err := podStore.FindPodcastsByRange(context.Background(), 0, 1)
	if err != nil {
		t.Fatalf("Test_FindPodcastRSS() error: %v", err)
	}
	require.NotEmpty(t, pods)
}

func Test_InsertPodcast(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	pod := &Podcast{ID: uuid.New(), Author: "Sam Schwartz", Description: "Test Insert Podcast", LinkURL: "https://syncapod.com/podcast", ImageURL: "http://syncapod.com/logo.png", Language: "en", Category: []int{1, 2, 3}, Explicit: "clean", RSSURL: "https://syncapod.com/podcast_test.rss"}
	err := podStore.InsertPodcast(context.Background(), pod)
	if err != nil {
		t.Fatalf("Test_InsertPodcast() error: %v", err)
	}
}

func Test_FindLatestEpisode(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	epi, err := podStore.FindLatestEpisode(context.Background(), testPod.ID)
	if err != nil {
		t.Fatalf("Test_FindLatestEpisode() error: %v", err)
	}
	require.Equal(t, testEpi2.ID, epi.ID)
}

func Test_FindEpisodeByID(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	epi, err := podStore.FindEpisodeByID(context.Background(), testEpi.ID)
	if err != nil {
		t.Fatalf("Test_FindEpisodeByID() error: %v", err)
	}
	require.Equal(t, testEpi.ID, epi.ID)
}

func Test_FindEpisodeByURL(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	epi, err := podStore.FindEpisodeByURL(context.Background(), testPod.ID, testEpi.EnclosureURL)
	if err != nil {
		t.Fatalf("Test_FindEpisodeByURL() error: %v", err)
	}
	require.Equal(t, testEpi.ID, epi.ID)
}

func Test_FindEpisodeNumber(t *testing.T) {
	podStore := NewPodcastStore(dbpg)

	epiFound, err := podStore.FindEpisodeNumber(context.Background(), testEpi.PodcastID, 0, 123)
	if err != nil {
		t.Fatalf("Test_FindEpisodeNumber() error finding episode: %v", err)
	}
	require.NotNil(t, *epiFound)
}

func Test_SearchPodcasts(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
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
	podStore := NewPodcastStore(dbpg)
	cats, err := podStore.FindAllCategories(context.Background())
	if err != nil {
		t.Fatalf("Test_FindEpisodeByURL() error: %v", err)
	}
	require.NotEmpty(t, cats)
}

func Test_FindUserEpisode(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	userEpi, err := podStore.FindUserEpisode(context.Background(), testUser.ID, testEpi.ID)
	if err != nil {
		t.Fatalf("Test_FindUserEpisode() error: %v", err)
	}
	require.NotNil(t, userEpi)
}

func Test_FindLastUserEpi(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	userEpi, err := podStore.FindLastUserEpi(context.Background(), testUser.ID)
	if err != nil {
		t.Fatalf("Test_FindLastUserEpi() error: %v", err)
	}
	require.NotNil(t, userEpi)
}

func Test_FindLastUserPlayedWithEpisode(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	userEpi, pod, epi, err := podStore.FindLastPlayed(context.Background(), testUser.ID)
	if err != nil {
		t.Fatalf("Test_FindLastUserEpiWithEpisode() error: %v", err)
	}
	require.NotEmpty(t, userEpi)
	require.NotEmpty(t, epi)
	require.NotEmpty(t, pod)
}

func Test_UpsertUserEpisode(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
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

func Test_FindEpisodesByRange(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	epis, err := podStore.FindEpisodesByRange(context.Background(), testPod.ID, 0, 2)
	if err != nil {
		t.Fatalf("Test_FindEpisodesByRange() error finding episodes: %v", err)
	}
	log.Println("pods:", epis)
	require.Equal(t, []Episode{*testEpi2, *testEpi}, epis)
}

func Test_InsAndDelSubscription(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	p := &Podcast{ID: uuid.New(), Category: []int{1, 2}}
	insertPodcastOrFail(podStore, p)
	s := &Subscription{UserID: testUser.ID, PodcastID: p.ID}
	err := podStore.InsertSubscription(context.Background(), s)
	if err != nil {
		t.Fatalf("Test_InsAndDelSubscription() error inserting subscription: %v", err)
	}
	err = podStore.DeleteSubscription(context.Background(), s.UserID, s.PodcastID)
	if err != nil {
		t.Fatalf("Test_InsAndDelSubscription() error deleting subscription: %v", err)
	}
}

func Test_FindSubscriptions(t *testing.T) {
	podStore := NewPodcastStore(dbpg)
	subs, err := podStore.FindSubscriptions(context.Background(), testUser.ID)
	if err != nil {
		t.Fatalf("Test_FindSubscriptions() error finding subscriptions: %v", err)
	}
	require.Equal(t, []Subscription{*testSub, *testSub2}, subs)
}

func TestPodcastStore_InsertCategory(t *testing.T) {
	type fields struct {
		db *pgxpool.Pool
	}
	type args struct {
		ctx context.Context
		cat *Category
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "insert_category",
			fields:  fields{dbpg},
			args:    args{ctx: context.Background(), cat: &Category{ID: 200, Name: "test cat", ParentID: 0}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PodcastStore{
				db: tt.fields.db,
			}
			if err := p.InsertCategory(tt.args.ctx, tt.args.cat); (err != nil) != tt.wantErr {
				t.Errorf("PodcastStore.InsertCategory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
