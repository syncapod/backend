package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sschwartz96/syncapod-backend/internal/auth"
	"github.com/sschwartz96/syncapod-backend/internal/db"
	"github.com/sschwartz96/syncapod-backend/internal/podcast"
	"github.com/sschwartz96/syncapod-backend/internal/util"
)

// Alexa intents events and directives
const (
	// Intents
	PlayPodcast       = "PlayPodcast"
	PlayLatestPodcast = "PlayLatestPodcast"
	PlayNthFromLatest = "PlayNthFromLatest"
	FastForward       = "FastForward"
	Rewind            = "Rewind"
	Pause             = "AMAZON.PauseIntent"
	Resume            = "AMAZON.ResumeIntent"

	// Events
	PlaybackNearlyFinished = "AudioPlayer.PlaybackNearlyFinished"
	PlaybackFinished       = "AudioPlayer.PlaybackFinished"

	// Directives
	DirPlay       = "AudioPlayer.Play"
	DirStop       = "AudioPlayer.Stop"
	DirClearQueue = "AudioPlayer.ClearQueue"
)

type AlexaHandler struct {
	auth auth.Auth
	pod  *podcast.PodController
	log  *slog.Logger
}

func CreateAlexaHandler(auth auth.Auth, podCon *podcast.PodController, log *slog.Logger) *AlexaHandler {
	return &AlexaHandler{
		auth: auth,
		pod:  podCon,
		log:  log,
	}
}

// Alexa handles all requests through /api/alexa endpoint
func (h *AlexaHandler) Alexa(res http.ResponseWriter, req *http.Request) {
	var resText, directive string

	body, err := io.ReadAll(req.Body)
	if err != nil {
		h.log.Warn("alexa: could not read the body of request", util.Err(err))
		// TODO: proper response here
		return
	}

	// audioplayer event or intent
	if strings.Contains(string(body), "\"even\"") {
		h.AudioEvent(res, req, body)
		return
	}

	var aData AlexaData
	err = json.Unmarshal(body, &aData)
	if err != nil {
		h.log.Warn("alexa: could not unmarshal json to object", util.Err(err))
		// TODO: proper response here
		return
	}

	// get the person or user accessToken
	token, err := getAccessToken(&aData)
	if err != nil {
		h.log.Warn("alexa: no accessToken", util.Err(err))
		resText = "No associated account, please link account in settings."
		// TODO: return early here?
	}

	// validate the token and return user
	userObj, err := h.auth.ValidateAccessToken(req.Context(), token)
	if err != nil {
		h.log.Warn("alexa: error validating access token", util.Err(err))
		resText = "Associated account has invalid token, please re-link account in settings."
	}
	// we have an error
	if resText != "" {
		aRes := createEmptyResponse(resText)
		aResJSON, _ := json.Marshal(&aRes)
		res.Write(aResJSON)
		return
	}

	name := aData.Request.Intent.AlexaSlots.Podcast.Value
	h.log.Debug("request name of podcast", slog.String("name", name))

	var response *AlexaResponseData
	var pod *db.Podcast
	var epi *db.Episode
	var offset int64

	h.log.Debug("request name of intent", slog.String("intent", aData.Request.Intent.Name))
	switch aData.Request.Intent.Name {
	case PlayPodcast:
		// search for the podcast given the name
		podcasts, err := h.pod.SearchPodcasts(req.Context(), name)
		if err != nil {
			resText = "Error occurred searching for podcast"
			break
		}
		// if the search came back with results defualt to first
		if len(podcasts) > 0 {
			pod = &podcasts[0]
			// either find latest episode or find the episode number
			eNumStr := aData.Request.Intent.AlexaSlots.Episode.Value
			if eNumStr != "" {
				epiNumber, err := strconv.Atoi(eNumStr)
				if err != nil {
					h.log.Warn("could not parse episode number", util.Err(err))
					resText = "Could not find episode, please try again."
					break
				}
				h.log.Debug("alexa play request", slog.Int("episode number", epiNumber))

				epi, err = h.pod.FindEpisodeNumber(req.Context(), pod.ID, 0, epiNumber)
				if err != nil {
					h.log.Warn("could not find episode with that number", slog.String("podcast title", pod.Title), slog.Int("episode number", epiNumber))
					resText = "Could not find episode with that number, please try again."
					break
				}
			} else {
				h.log.Debug("finding latest episode of", slog.String("podcast title", pod.Title))
				epi, err = h.pod.FindLatestEpisode(req.Context(), pod.ID)
				if err != nil {
					h.log.Warn("latest episode could not be found", slog.String("podcast title", pod.Title), util.Err(err))
					resText = "Could not find episode, please try again."
					break
				}
			}

			directive = DirPlay
		} else {
			resText = "Podcast of the name: " + name + ", not found"
		}

	case PlayNthFromLatest:

	case FastForward:
		directive = DirPlay
		pod, epi, resText, offset = h.moveAudio(req.Context(), &aData, true)

	case Rewind:
		directive = DirPlay
		pod, epi, resText, offset = h.moveAudio(req.Context(), &aData, false)

	case Pause:
		audioTokens := strings.Split(aData.Context.AudioPlayer.Token, ";")
		// TODO: add debug logging: slog.Debug("audioplayer tkn:", aData.Context.AudioPlayer.Token)
		if len(audioTokens) > 1 {
			//podID := uuid.MustParse(audioTokens[1])
			epiID := uuid.MustParse(audioTokens[2])
			directive = DirStop
			// TODO: handle error better back to user
			go func() {
				err := h.pod.PodcastStore.UpsertUserEpisode(
					context.Background(),
					&db.UserEpisode{UserID: userObj.ID, EpisodeID: epiID,
						OffsetMillis: aData.Context.AudioPlayer.OffsetInMilliseconds,
						Played:       false,
					},
				)
				if err != nil {
					h.log.Warn("alexa pause, error updating offset", util.Err(err))
				}
			}()
		} else {
			resText = "Please play a podcast first"
		}

	case Resume:
		splitID := strings.Split(aData.Context.AudioPlayer.Token, ";")
		if len(splitID) > 1 {
			podID := uuid.MustParse(splitID[1])
			epiID := uuid.MustParse(splitID[2])
			pod, err = h.pod.FindPodcastByID(req.Context(), podID)
			if err != nil {
				h.log.Warn("could not find podcast with given ID", slog.String("podcast id", podID.String()), util.Err(err))
				resText = "Please try playing new podcast"
				break
			}
			epi, err = h.pod.FindEpisodeByID(req.Context(), epiID)
		} else {
			// need to get episode and user episode
			userEpi := &db.UserEpisode{}
			userEpi, pod, epi, err = h.pod.FindLastPlayed(req.Context(), userObj.ID)
			offset = userEpi.OffsetMillis
			if err != nil {
				h.log.Warn("could not find podcast the user last played", slog.String("user id", userObj.ID.String()), util.Err(err))
				resText = "Couldn't find any currently played podcast, please play new one"
				break
			}
		}

		if epi != nil {
			directive = DirPlay
			resText = "Resuming"
			if offset == 0 {
				// we want to update the offset via database, so we are sure we have the latest update
				//offset = aData.Context.AudioPlayer.OffsetInMilliseconds
			}
		} else {
			resText = "Episode not found, please try playing new podcast"
		}

	default:
		resText = "This command is currently not supported, please request"
	}

	// If we are creating an alexa audio repsonse
	if directive != "" {
		// get details from non-nil episode
		if userObj != nil && pod != nil && epi != nil {
			if resText == "" {
				resText = "Playing " + pod.Title + ", " + epi.Title
			}
			if offset == 0 {
				userEpi, err := h.pod.FindUserEpisode(req.Context(), userObj.ID, epi.ID)
				// either the userEpi was not found or there was an error
				if err != nil {
					offset = 0
					// TODO: handle internal server error
				} else {
					offset = userEpi.OffsetMillis
				}
			}
			h.log.Debug("alexa: play request", slog.Int64("offset", offset))
			response = createAudioResponse(directive, userObj.ID.String(),
				resText, pod, epi, offset)
		} else {
			response = createPauseResponse(directive)
		}
	} else {
		response = createEmptyResponse(resText)
	}

	jsonRes, err := json.Marshal(response)
	if err != nil {
		h.log.Error("could not marshal alexa repsonse", util.Err(err))
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonRes)
}

// moveAudio takes pointer to aData and bool for direction
// returns pointers to podcast and episode, response text and offset in millis
func (h *AlexaHandler) moveAudio(ctx context.Context, aData *AlexaData, forward bool) (*db.Podcast, *db.Episode, string, int64) {
	var pod *db.Podcast
	var epi *db.Episode
	var resText string
	var offset int64
	var err error

	audioTokens := strings.Split(aData.Context.AudioPlayer.Token, ";")
	if len(audioTokens) > 1 {
		pID := uuid.MustParse(audioTokens[1])
		eID := uuid.MustParse(audioTokens[2])

		// find podcast
		pod, err = h.pod.FindPodcastByID(ctx, pID)
		if err != nil {
			h.log.Error("error finding podcast", util.Err(err))
			resText = "Error occurred, please try again"
			return nil, nil, resText, 0
		}

		// find episode
		epi, err = h.pod.FindEpisodeByID(ctx, eID)
		if err != nil {
			h.log.Error("error finding episode", util.Err(err))
			resText = "Error occurred, please try again"
			return nil, nil, resText, 0
		}

		// get the current time and duration to move
		curTime := aData.Context.AudioPlayer.OffsetInMilliseconds
		dura := convertISO8601ToMillis(aData.Request.Intent.AlexaSlots.Duration.Value)
		durString := durationToText(time.Millisecond * time.Duration(dura))

		h.log.Debug("alexa move",
			slog.Int64("current time", curTime),
			slog.Any("aData duration", aData.Request.Intent.AlexaSlots.Duration.Value),
			slog.Int64("duraiton calculated", dura),
			slog.String("durString", durString),
		)

		h.log.Error("error finding episode", util.Err(err))

		if forward {
			offset = curTime + dura
			resText = "Fast-forwarded " + durString
		} else {
			offset = curTime - dura
			resText = "Rewound " + durString
		}

		if offset < 0 {
			offset = 1
		} else {
			// check if we are trying to fast forward past end of episode
			if epi.Duration < offset {
				tilEnd := time.Duration(epi.Duration-curTime) * time.Millisecond
				resText = "Cannot fast forward further than: " + durationToText(tilEnd)
				offset = curTime
			}
		}
	} else {
		resText = "Please play a podcast first"
	}

	return pod, epi, resText, offset
}

func durationToText(dur time.Duration) string {
	bldr := strings.Builder{}
	if int(dur.Hours()) == 1 {
		bldr.WriteString("1 hour, ")
	} else if dur.Hours() > 1 {
		bldr.WriteString(strconv.Itoa(int(dur.Hours())))
		bldr.WriteString(" hours, ")
	}
	dur = dur - dur.Truncate(time.Hour)

	if int(dur.Minutes()) == 1 {
		bldr.WriteString("1 minute, ")
	} else if dur.Minutes() > 1 {
		bldr.WriteString(strconv.Itoa(int(dur.Minutes())))
		bldr.WriteString(" minutes, ")
	}
	dur = dur - dur.Truncate(time.Minute)

	if int(dur.Seconds()) == 1 {
		bldr.WriteString("1 second, ")
	} else if dur.Seconds() > 1 {
		bldr.WriteString(strconv.Itoa(int(dur.Seconds())))
		bldr.WriteString(" seconds, ")
	}

	return bldr.String()
}

func createAudioResponse(directive, userID, text string,
	pod *db.Podcast, epi *db.Episode, offset int64) *AlexaResponseData {

	mp3URL := epi.EnclosureURL
	if !strings.Contains(mp3URL, "https") {
		mp3URL = strings.Replace(mp3URL, "http", "https", 1)
	}

	imgURL := epi.ImageURL
	if imgURL == "" {
		imgURL = pod.ImageURL
		if imgURL == "" {
			// custom generic defualt image
			// TODO: own custom image
			imgURL = "https://emby.media/community/uploads/inline/355992/5c1cc71abf1ee_genericcoverart.jpg"
		}
	}

	return &AlexaResponseData{
		Version: "1.0",
		Response: AlexaResponse{
			Directives: []AlexaDirective{
				{
					Type:         directive,
					PlayBehavior: "REPLACE_ALL",
					AudioItem: AlexaAudioItem{
						Stream: AlexaStream{
							URL:                  mp3URL,
							Token:                userID + ";" + pod.ID.String() + ";" + epi.ID.String(),
							OffsetInMilliseconds: offset,
						},
						Metadata: AlexaMetadata{
							Title:    epi.Title,
							Subtitle: epi.Summary,
							Art: AlexaArt{
								Sources: []AlexaURL{
									{
										URL:    imgURL,
										Height: 144,
										Width:  144,
									},
								},
							},
						},
					},
				},
			},
			OutputSpeech: AlexaOutputSpeech{
				Type: "PlainText",
				Text: text,
			},
			ShouldEndSession: true,
		},
	}
}

func createPauseResponse(directive string) *AlexaResponseData {
	return &AlexaResponseData{
		Version: "1.0",
		Response: AlexaResponse{
			Directives: []AlexaDirective{
				{
					Type: directive,
				},
			},
			OutputSpeech: AlexaOutputSpeech{
				Type: "PlainText",
				Text: "Paused",
			},
			ShouldEndSession: true,
		},
	}
}

func createEmptyResponse(text string) *AlexaResponseData {
	return &AlexaResponseData{
		Version: "1.0",
		Response: AlexaResponse{
			Directives: nil,
			OutputSpeech: AlexaOutputSpeech{
				Type:         "PlainText",
				Text:         text,
				PlayBehavior: "REPLACE_ENQUEUE",
			},
			ShouldEndSession: true,
		},
	}
}

func convertISO8601ToMillis(data string) int64 {
	data = data[2:]

	var durRegArr [3]*regexp.Regexp
	var durStrArr [3]string
	var durIntArr [3]int64

	durRegArr[0], _ = regexp.Compile("([0-9]+)H")
	durRegArr[1], _ = regexp.Compile("([0-9]+)M")
	durRegArr[2], _ = regexp.Compile("([0-9]+)S")

	for i := range durStrArr {
		durStrArr[i] = durRegArr[i].FindString(data)
		if len(durStrArr[i]) > 1 {
			str := durStrArr[i]
			val, _ := strconv.Atoi(str[:len(str)-1])
			durIntArr[i] = int64(val)
		}
	}

	return (durIntArr[0])*int64(3600000) +
		(durIntArr[1])*int64(60000) +
		(durIntArr[2])*int64(1000)
}

// getIDsFromToken takes token string and returns (userID,podID,epiID,error)
// returns error if the token is malformed
func getIDsFromToken(token string) (string, string, string, error) {
	// token is in this format userid-podid-epiid
	split := strings.Split(token, "-")
	if len(split) != 3 {
		return "", "", "", errors.New("not valid playback token")
	}
	return split[0], split[1], split[2], nil
}

func getAccessToken(data *AlexaData) (string, error) {
	if data.Context.System.Person.AccessToken != "" {
		return data.Context.System.Person.AccessToken, nil
	} else if data.Context.System.User.AccessToken != "" {
		return data.Context.System.User.AccessToken, nil
	}
	return "", errors.New("no accessToken")
}

// AudioEvent handles responses from the Alexa audioplayer
func (h *AlexaHandler) AudioEvent(res http.ResponseWriter, req *http.Request, body []byte) {
	var data AudioData
	err := json.Unmarshal(body, &data)
	if err != nil {
		h.log.Error("failed to unmarshal audio event", util.Err(err))
		return
	}

	uID, pID, eID, err := getIDsFromToken(data.Event.Payload.Token)
	if err != nil {
		h.log.Error("error retrieving user, podcast, or episode item from token", util.Err(err))
		return
	}

	userID, err := uuid.Parse(uID)
	if err != nil {
		h.log.Error("error parsing user id", util.Err(err))
		return
	}

	podID, err := uuid.Parse(pID)
	if err != nil {
		h.log.Error("error parsing podcast id", util.Err(err))
		return
	}

	epiID, err := uuid.Parse(eID)
	if err != nil {
		h.log.Error("error parsing episode id", util.Err(err))
		return
	}

	h.log.Debug("audio event",
		slog.String("name", data.Event.Header.Name),
		slog.String("userID", userID.String()),
		slog.String("podID", podID.String()),
		slog.String("userID", epiID.String()),
	)

	switch data.Event.Header.Name {
	case PlaybackNearlyFinished:
		return
	case PlaybackFinished:
		err := h.pod.UpsertUserEpisode(req.Context(), &db.UserEpisode{EpisodeID: epiID, UserID: userID, Played: true, LastSeen: time.Unix(0, 0)})
		if err != nil {
			h.log.Error("failed to update the userEpi as played", util.Err(err))
		}
	}
}

// AlexaData contains all the informatino and data from request sent from alexa
type AlexaData struct {
	Version string       `json:"version,omitempty"`
	Context AlexaContext `json:"context,omitempty"`
	Request AlexaRequest `json:"request,omitempty"`
}

// AlexaContext contains system
type AlexaContext struct {
	System      AlexaSystem      `json:"System,omitempty"`
	AudioPlayer AlexaAudioPlayer `json:"AudioPlayer,omitempty"`
}

// AlexaSystem is the container for person and user
type AlexaSystem struct {
	Person AlexaPerson `json:"person,omitempty"`
	User   AlexaUser   `json:"user,omitempty"`
}

// AlexaAudioPlayer contains info of the currently played track if available
type AlexaAudioPlayer struct {
	OffsetInMilliseconds int64  `json:"offsetInMilliseconds,omitempty"`
	Token                string `json:"token,omitempty"`
	PlayActivity         string `json:"playActivity,omitempty"`
}

// AlexaPerson holds the info about the person who explicitly called the skill
type AlexaPerson struct {
	PersonID    string `json:"personId,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}

// AlexaUser contains info about the user that holds the skill
type AlexaUser struct {
	UserID      string `json:"userId,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}

// AlexaRequest holds all the information and data
type AlexaRequest struct {
	Type                 string      `json:"type,omitempty"`
	RequestID            string      `json:"requestId,omitempty"`
	Timestamp            time.Time   `json:"timestamp,omitempty"`
	Token                string      `json:"token,omitempty"`
	OffsetInMilliseconds int64       `json:"offsetInMilliseconds,omitempty"`
	Intent               AlexaIntent `json:"intent,omitempty"`
}

// AlexaIntent holds information and data of intent sent from alexa
type AlexaIntent struct {
	Name       string     `json:"name,omitempty"`
	AlexaSlots AlexaSlots `json:"slots,omitempty"`
}

// AlexaSlots are the container for the slots
type AlexaSlots struct {
	Nth      AlexaSlot `json:"nth,omitempty"`
	Episode  AlexaSlot `json:"episode,omitempty"`
	Podcast  AlexaSlot `json:"podcast,omitempty"`
	Duration AlexaSlot `json:"duration,omitempty"`
}

// AlexaSlot holds information of the slot for the intent
type AlexaSlot struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// AlexaResponseData contains the version and response
type AlexaResponseData struct {
	Version  string        `json:"version,omitempty"`
	Response AlexaResponse `json:"response,omitempty"`
}

// AlexaResponse contains the actual response
type AlexaResponse struct {
	Directives       []AlexaDirective  `json:"directives,omitempty"`
	OutputSpeech     AlexaOutputSpeech `json:"outputSpeech,omitempty"`
	ShouldEndSession bool              `json:"shouldEndSession,omitempty"`
}

// AlexaDirective tells alexa what to do
type AlexaDirective struct {
	Type         string         `json:"type,omitempty"`
	PlayBehavior string         `json:"playBehavior,omitempty"`
	AudioItem    AlexaAudioItem `json:"audioItem,omitempty"`
}

// AlexaAudioItem holds information of audio track
type AlexaAudioItem struct {
	Stream   AlexaStream   `json:"stream,omitempty"`
	Metadata AlexaMetadata `json:"metadata,omitempty"`
}

// AlexaStream contains information about the audio url and offset
type AlexaStream struct {
	Token                string `json:"token,omitempty"`
	URL                  string `json:"url,omitempty"`
	OffsetInMilliseconds int64  `json:"offsetInMilliseconds,omitempty"`
}

// AlexaMetadata contains information about the stream
type AlexaMetadata struct {
	Title    string   `json:"title,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	Art      AlexaArt `json:"art,omitempty"`
}

// AlexaArt contains info for album art of stream
type AlexaArt struct {
	Sources []AlexaURL `json:"sources,omitempty"`
}

// AlexaURL is the container for AlexaArt
type AlexaURL struct {
	URL    string `json:"url,omitempty"`
	Height int    `json:"height,omitempty"`
	Width  int    `json:"width,omitempty"`
}

// AlexaOutputSpeech takes type: "PlainText", text, and playBehavior: REPLACE_ENQUEUE
type AlexaOutputSpeech struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	PlayBehavior string `json:"playBehavior,omitempty"`
}

// AudioData is the container for AudioEvent
type AudioData struct {
	Event AudioEvent `json:"event,omitempty"`
}

// AudioEvent is the container for audioplayer response
type AudioEvent struct {
	Header          AudioHeader   `json:"header,omitempty"`
	Payload         AudioPayload  `json:"payload,omitempty"`
	PlaybackReports []AudioReport `json:"playbackReports,omitempty"`
}

// AudioHeader contains header info of AudioEvent
type AudioHeader struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	MessageID string `json:"messageId,omitempty"`
}

// AudioPayload contains the main info of AudioEvent
type AudioPayload struct {
	Token                string          `json:"token,omitempty"`
	OffsetInMilliseconds int64           `json:"offsetInMilliseconds,omitempty"`
	PlaybackAttributes   AudioAttributes `json:"playbackAttributes,omitempty"`
}

// AudioAttributes contains the attributes of the AudioPayload & AudioReport
type AudioAttributes struct {
	Name                    string `json:"name,omitempty"`
	Codec                   string `json:"codec,omitempty"`
	SamplingRateInHertz     int64  `json:"samplingRateInHertz,omitempty"`
	DataRateInBitsPerSecond int64  `json:"dataRateInBitsPerSecond,omitempty"`
}

// AudioReport contains playback info for AudioEvent
type AudioReport struct {
	StartOffsetInMilliseconds string          `json:"startOffsetInMilliseconds,omitempty"`
	EndOffsetInMilliseconds   string          `json:"endOffsetInMilliseconds,omitempty"`
	PlaybackAttributes        AudioAttributes `json:"playbackAttributes,omitempty"`
}
