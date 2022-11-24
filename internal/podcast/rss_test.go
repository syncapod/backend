package podcast

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/stretchr/testify/require"
)

func Test_RSS(t *testing.T) {
	podStore := db.NewPodcastStore(dbpg)
	podController, err := NewPodController(podStore)
	if err != nil {
		t.Fatalf("Test_RSS error setting up: %v", err)
	}
	rssController := NewRSSController(podController)
	//rssURL := "https://changelog.com/gotime/feed"
	rssURL := "https://feeds.twit.tv/twit.xml"
	u, _ := url.Parse(rssURL)

	// test add the podcast
	pod, err := rssController.AddPodcast(context.Background(), u)
	if err != nil {
		t.Fatalf("Test_RSS() error adding new podcast: %v", err)
	}
	podID := &pod.ID

	// get the latest episode
	epi, err := rssController.podController.FindLatestEpisode(context.Background(), *podID)
	if err != nil {
		t.Fatalf("Test_RSS() error finding latest episode: %v", err)
	}

	// delete the last episode to air
	_, err = dbpg.Exec(context.Background(),
		`DELETE FROM Episodes
		WHERE id=any(array(SELECT id FROM Episodes ORDER BY pub_date DESC LIMIT 1))`)
	if err != nil {
		t.Fatalf("Test_RSS() error deleting latest episode %v", err)
	}

	// test update the podcast (adds the last episode)
	err = rssController.UpdatePodcasts()
	if err != nil {
		t.Fatalf("Test_RSS() error updating podcasts: %v", err)
	}

	// find the latest, compare to previous latest
	epi2, err := rssController.podController.FindLatestEpisode(context.Background(), *podID)
	if err != nil {
		t.Fatalf("Test_RSS() error finding latest episode(2): %v", err)
	}
	epi2.ID = epi.ID
	require.Equal(t, *epi, *epi2)
}

func Test_parseDuration(t *testing.T) {
	type args struct {
		d string
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "seconds",
			args: args{
				d: "5400",
			},
			want: 5400000,
		},
		{
			name: "hh:mm:ss",
			args: args{
				d: "01:30:30",
			},
			want: 5430000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseDuration(tt.args.d)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseRFC2822(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				s: "Thu, 08 Oct 2020 15:30:00 +0000",
			},
			want:    time.Unix(1602171000, 0),
			wantErr: false,
		},
		{
			name: "2",
			args: args{
				s: "Tue, 06 Oct 2020 20:00:00 PDT",
			},
			want:    time.Unix(1602039600, 0),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRFC2822ToUTC(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRFC2822() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got.UTC(), tt.want.UTC()) {
				t.Errorf("parseRFC2822() = %v, want %v", got.UTC(), tt.want.UTC())
			}
		})
	}
}
