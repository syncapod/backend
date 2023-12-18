package podcast

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

type RSSController struct {
	podController *PodController
	log           *slog.Logger
}

func NewRSSController(podController *PodController, log *slog.Logger) *RSSController {
	return &RSSController{
		podController: podController,
		log:           log,
	}
}

var tzMap = map[string]string{
	"PST": "-0800", "PDT": "-0700",
	"MST": "-0700", "MDT": "-0600",
	"CST": "-0600", "CDT": "-0500",
	"EST": "-0500", "EDT": "-0400",
}

// UpdatePodcasts attempts to go through the list of podcasts update them via RSS feed
func (c *RSSController) UpdatePodcasts() error {
	var podcasts []db.Podcast
	var err error
	// just increments start and end indices
	for start, end := 0, 10; ; start, end = end, end+10 {
		podcasts, err = c.podController.FindPodcastsByRange(context.Background(), start, end)
		if err != nil || len(podcasts) == 0 {
			break // will eventually break
		}
		var wg sync.WaitGroup
		for i := range podcasts {
			pod := &podcasts[i]
			wg.Add(1)
			go func() {
				c.log.Info("starting UpdatePodcast()", slog.String("podcast title", pod.Title))
				err = c.updatePodcast(pod)
				if err != nil {
					c.log.Error("UpdatePodcasts() error updating podcast", slog.Any("podcast", pod), util.Err(err))
				}
				c.log.Info("finished updatePodcast():", slog.String("podcast title", pod.Title))
				wg.Done()
			}()
		}
		wg.Wait()
	}
	if err != nil {
		return fmt.Errorf("UpdatePodcasts() error retrieving from db: %v", err)
	}
	return nil
}

// updatePodcast updates the given podcast via RSS feed
func (c *RSSController) updatePodcast(pod *db.Podcast) error {
	// get rss from url
	rssResp, err := DownloadRSS(pod.RSSURL)
	if err != nil {
		return fmt.Errorf("updatePodcast() error downloading rss: %v", err)
	}
	// defer closing
	defer func() {
		err := rssResp.Close()
		if err != nil {
			c.log.Error("updatePodcast() error closing rss response:", util.Err(err))
		}
	}()
	// parse rss from respone.Body
	newPod, err := parseRSS(rssResp)
	if err != nil {
		return fmt.Errorf("updatePodcast() error parsing RSS: %v", err)
	}

	for e := range newPod.Channel.Items {
		epi := rssItemToDBEpisode(&newPod.Channel.Items[e], pod.ID, c.log)
		// check if the latest episode is in collection
		exists := c.podController.DoesEpisodeExist(context.Background(), pod.ID, epi.EnclosureURL)
		if !exists {
			err = c.podController.InsertEpisode(context.Background(), epi)
			if err != nil {
				return fmt.Errorf("updatePodcast() error upserting episode: %v", err)
			}
		}
	}
	return nil
}

// AddNewPodcast takes RSS url and a reader to the RSS feed and
// inserts the podcast and its episodes into the db
// returns error if podcast already exists
func (c *RSSController) AddNewPodcast(url string, r io.Reader) (*db.Podcast, error) {
	// check if podcast already contains that rss url
	exists := c.podController.DoesPodcastExist(context.Background(), url)
	if exists {
		return nil, errors.New("AddNewPodcast() podcast already exists")
	}

	// parse rssPod
	rssPod, err := parseRSS(r)
	if err != nil {
		return nil, err
	}
	pod, err := c.rssChannelToPodcast(&rssPod.Channel, uuid.New(), url)
	if err != nil {
		return nil, fmt.Errorf("AddNewPodcast() error converting rss: %v", err)
	}

	// insert podcast
	err = c.podController.InsertPodcast(context.Background(), pod)
	if err != nil {
		return nil, fmt.Errorf("AddNewPodcast() error adding new podcast: %v", err)
	}

	// loop through episodes and save them
	for i := range rssPod.Channel.Items {
		epi := rssItemToDBEpisode(&rssPod.Channel.Items[i], pod.ID, c.log)
		err = c.podController.InsertEpisode(context.Background(), epi)
		if err != nil {
			c.log.Error("AddNewPodcast() couldn't insert episode: ", util.Err(err))
		}
	}
	return pod, nil
}

func DownloadRSS(url string) (io.ReadCloser, error) {
	http.DefaultClient.Timeout = time.Second * 5
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("DownloadRSS() error: %v", err)
	}
	return resp.Body, nil
}

// parseRSS takes in reader path and unmarshals the data
func parseRSS(r io.Reader) (*rss, error) {
	// set up rssFeed feed object and decoder
	rssFeed := &rss{}
	decoder := xml.NewDecoder(r)
	decoder.DefaultSpace = "Default"
	// decode
	err := decoder.Decode(rssFeed)
	if err != nil {
		return nil, fmt.Errorf("rss.parseRSS() error decoding rss: %v", err)
	}
	return rssFeed, nil
}

func findTimezoneOffset(tz string) (string, error) {
	offset, ok := tzMap[tz]
	if !ok {
		return "", errors.New("timezone not found")
	}
	return offset, nil
}

// parseRFC2822ToUTC parses the string in RFC2822 date format
// returns pointer to time object and error
// returns time.Now() even if error occurs
func parseRFC2822ToUTC(s string) (*time.Time, error) {
	if s == "" {
		t := time.Now()
		return &t, fmt.Errorf("parseRFC2822ToUTC() no time provided")
	}
	if !strings.Contains(s, "+") && !strings.Contains(s, "-") {
		fields := strings.Fields(s)
		tz := fields[len(fields)-1]
		offset, err := findTimezoneOffset(tz)
		if err != nil {
			t := time.Now()
			return &t, err
		}
		s = strings.ReplaceAll(s, tz, offset)
	}
	t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", s)
	if err != nil {
		return &t, err
	}
	return &t, nil
}

// parseDuration takes in the string duration and returns the duration in millis
func parseDuration(d string) (int64, error) {
	if d == "" {
		return 0, fmt.Errorf("parseDuration() error empty duration string")
	}
	// check if they just applied the seconds
	if !strings.Contains(d, ":") {
		sec, err := strconv.Atoi(d)
		if err != nil {
			return 0, fmt.Errorf("parseDuration() error converting duration of episode: %v", err)
		}
		return int64(sec) * int64(1000), nil
	}
	var millis int64
	multiplier := int64(1000)

	// format hh:mm:ss || mm:ss
	split := strings.Split(d, ":")

	for i := len(split) - 1; i >= 0; i-- {
		v, _ := strconv.Atoi(split[i])
		millis += int64(v) * multiplier
		multiplier *= int64(60)
	}

	return millis, nil
}

type rss struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string `xml:"title"`
	Copyright   string `xml:"copyright"`
	Link        string `xml:"link"`
	Language    string `xml:"language"`
	Description string `xml:"description"`
	Author      string `xml:"author"`
	Summary     string `xml:"summary"`
	Explicit    string `xml:"explicit"`
	Type        string `xml:"type"`
	Complete    string `xml:"complete"`
	Block       string `xml:"block"`
	PubDate     string `xml:"pubDate"`
	Image       struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
	} `xml:"image"`
	Keywords string `xml:"keywords"`
	Owner    struct {
		Text  string `xml:",chardata"`
		Name  string `xml:"name"`
		Email string `xml:"email"`
	} `xml:"owner"`
	Categories []Category `xml:"category"`
	Items      []rssItem  `xml:"item"`
}

type rssItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	Guid  struct {
		Text        string `xml:",chardata"`
		IsPermaLink string `xml:"isPermaLink,attr"`
	} `xml:"guid"`
	PubDate   string `xml:"pubDate"`
	Enclosure struct {
		URL    string `xml:"url,attr"`
		Length string `xml:"length,attr"`
		Type   string `xml:"type,attr"`
	} `xml:"enclosure"`
	Description string `xml:"description"`
	Encoded     string `xml:"encoded"`
	EpisodeType string `xml:"episodeType"`
	Episode     string `xml:"episode"`
	Season      string `xml:"season"`
	Image       struct {
		Href  string `xml:"href,attr"`
		Title string `xml:"title,attr"`
	} `xml:"image"`
	Duration string `xml:"duration"`
	Explicit string `xml:"explicit"`
	Keywords string `xml:"keywords"`
	Subtitle string `xml:"subtitle"`
	Summary  string `xml:"summary"`
	Creator  string `xml:"creator"`
	Author   string `xml:"author"`
}

type Category struct {
	ID            int
	Name          string     `xml:"text,attr"`
	Subcategories []Category `xml:"category"`
}

func (c *RSSController) rssChannelToPodcast(r *rssChannel, id uuid.UUID, rssURL string) (*db.Podcast, error) {
	pubDate, err := parseRFC2822ToUTC(r.PubDate)
	if err != nil {
		c.log.Error("rssChannelToPodcast() error converting pubdate:", util.Err(err))
	}
	cats, err := c.podController.catCache.TranslateCategories(r.Categories)
	if err != nil {
		return nil, fmt.Errorf("rssChannelToPodcast() error translating categories: %v", err)
	}
	return &db.Podcast{
		ID:          id,
		Title:       r.Title,
		Description: r.Description,
		ImageURL:    r.Image.Href,
		Language:    r.Language,
		Category:    cats,
		Explicit:    r.Explicit,
		Author:      r.Author,
		LinkURL:     r.Link,
		OwnerName:   r.Owner.Name,
		OwnerEmail:  r.Owner.Email,
		Episodic:    r.Type == "episodic",
		Copyright:   r.Copyright,
		Block:       strings.ToLower(r.Block) == "yes",
		Complete:    strings.ToLower(r.Complete) == "yes",
		PubDate:     *pubDate,
		Keywords:    r.Keywords,
		Summary:     r.Summary,
		RSSURL:      rssURL,
	}, nil
}

func rssItemToDBEpisode(r *rssItem, podID uuid.UUID, log *slog.Logger) *db.Episode {
	enclosureLen, err := strconv.ParseInt(r.Enclosure.Length, 10, 64)
	if err != nil {
		log.Error("rssItemToDBEpisode() error parsing enclosure length:", util.Err(err))
	}
	pubDate, err := parseRFC2822ToUTC(r.PubDate)
	if err != nil {
		log.Error("rssItemToDBEpisode() error converting pubdate:", util.Err(err))
	}
	duration, err := parseDuration(r.Duration)
	if err != nil {
		log.Error("rssItemToDBEpisode() error parsing duration:", util.Err(err))
	}
	episode, _ := strconv.Atoi(r.Episode)
	season, _ := strconv.Atoi(r.Season)

	return &db.Episode{
		ID:              uuid.New(),
		Title:           r.Title,
		EnclosureURL:    r.Enclosure.URL,
		EnclosureLength: enclosureLen,
		EnclosureType:   r.Enclosure.Type,
		PubDate:         *pubDate,
		Description:     r.Description,
		Duration:        duration,
		LinkURL:         r.Link,
		ImageURL:        r.Image.Href,
		ImageTitle:      r.Image.Title,
		Explicit:        r.Explicit,
		Episode:         episode,
		Season:          season,
		EpisodeType:     r.EpisodeType,
		Summary:         r.Summary,
		Subtitle:        r.Subtitle,
		Encoded:         r.Encoded,
		PodcastID:       podID,
	}
}
