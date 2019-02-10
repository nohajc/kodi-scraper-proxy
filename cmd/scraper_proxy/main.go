package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	json "github.com/virtuald/go-ordered-json"

	"github.com/elazarl/goproxy"
)

// ScraperAdapter in an interface for kodi scraper filters
// which can modify response from the scraper source
type ScraperAdapter interface {
	Host() string
	ResponseFilter(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response
}

// EpisodeOrderingMap is an interface for mapping episode numbers to different numbers
type EpisodeOrderingMap interface {
	// HasSpecialOrdering returns true if episodes of the given show need reordering
	HasSpecialOrdering(showID int64) bool
	// FromProductionToAired takes episode number in production order
	// and returns the corresponding episode number in aired order
	FromProductionToAired(showID int64, season int, episode int) (int, int)
}

// NewProxyWithScraperAdapters returns new HTTP proxy server with the given scraper adapters
func NewProxyWithScraperAdapters(adapters ...ScraperAdapter) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		var reqBodyBuf bytes.Buffer
		io.Copy(&reqBodyBuf, ctx.Req.Body)
		reqBody, _ := ioutil.ReadAll(&reqBodyBuf)
		log.Printf("Request: %s, %v\n", ctx.Req.URL.Path, reqBody)

		for _, adp := range adapters {
			if ctx.Req.Host == adp.Host() {
				return adp.ResponseFilter(resp, ctx)
			}
		}

		return resp
	})

	return proxy
}

// Episode structure inside JSON response
type Episode struct {
	AirDate        string               `json:"air_date"`
	EpisodeNumber  int                  `json:"episode_number"`
	ID             int64                `json:"id"`
	Name           string               `json:"name"`
	Overview       string               `json:"overview"`
	ProductionCode string               `json:"production_code"`
	SeasonNumber   int                  `json:"season_number"`
	ShowID         int64                `json:"show_id"`
	StillPath      string               `json:"still_path"`
	VoteAverage    float64              `json:"vote_average"`
	VoteCount      int                  `json:"vote_count"`
	Crew           []json.OrderedObject `json:"crew"`
	GuestStars     []json.OrderedObject `json:"guest_stars"`
}

// ScraperResponse is a parsed JSON response from scraper source
type ScraperResponse struct {
	UnID         string    `json:"_id"`
	AirDate      string    `json:"air_date"`
	Episodes     []Episode `json:"episodes"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	ID           int64     `json:"id"`
	PosterPath   string    `json:"poster_path"`
	SeasonNumber int       `json:"season_number"`
	//Images       []json.OrderedObject `json:"images"`
}

// TMDBScraperOrderingAdapter changes ordering of episodes based on provided mapping (TODO)
type TMDBScraperOrderingAdapter struct {
	orderingMap EpisodeOrderingMap
}

// Host returns host name of the scraper source
func (*TMDBScraperOrderingAdapter) Host() string {
	return "api.themoviedb.org"
}

// ResponseFilter modifies response from the scraper source to apply the new ordering
func (adp *TMDBScraperOrderingAdapter) ResponseFilter(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if !strings.Contains(ctx.Req.URL.Path, "/season/") {
		return resp
	}

	var tvShowID int64
	var seasonNum int
	n, err := fmt.Sscanf(ctx.Req.URL.Path, "/3/tv/%d/season/%d", &tvShowID, &seasonNum)
	if n != 2 || err != nil {
		log.Printf("Error parsing season number from request path: %v\n", err)
		return resp
	}

	log.Printf("Requested TV show %d, season %d\n", tvShowID, seasonNum)

	if !adp.orderingMap.HasSpecialOrdering(tvShowID) {
		log.Println("Doesn't need reordering")
		return resp
	}

	log.Println("Needs reordering")

	dec := json.NewDecoder(resp.Body)
	dec.UseOrderedObject()
	parsedResponse := ScraperResponse{}
	err = dec.Decode(&parsedResponse)
	if err != nil {
		log.Printf("Error parsing JSON from response: %v\n", err)
		return resp
	}

	episodesReordered := make([]Episode, len(parsedResponse.Episodes))
	for i := range parsedResponse.Episodes {
		_, airedEpNum := adp.orderingMap.FromProductionToAired(tvShowID, seasonNum, i+1)
		episodesReordered[i] = parsedResponse.Episodes[airedEpNum-1]
		episodesReordered[i].EpisodeNumber = i + 1
	}
	parsedResponse.Episodes = episodesReordered

	var newBodyBuf bytes.Buffer
	enc := json.NewEncoder(&newBodyBuf)

	err = enc.Encode(parsedResponse)
	if err != nil {
		log.Printf("Error serializing modified response: %v\n", err)
		return resp
	}

	resp.Body = ioutil.NopCloser(&newBodyBuf)
	return resp
}

// OfflineOrderingMap contains episode ordering map loaded from an offline resource
type OfflineOrderingMap struct {
	Table map[int64]map[int][]string
}

// NewOfflineOrderingMap creates new instance from Config
func NewOfflineOrderingMap(cfg Config) *OfflineOrderingMap {
	result := &OfflineOrderingMap{make(map[int64]map[int][]string)}

	for _, tvShow := range cfg.TVShows {
		result.Table[tvShow.ID] = make(map[int][]string)
		for _, epSeasonMap := range tvShow.Ordering {
			result.Table[tvShow.ID][epSeasonMap.Season] = make([]string, len(epSeasonMap.Episodes))
			copy(result.Table[tvShow.ID][epSeasonMap.Season], epSeasonMap.Episodes)
		}
	}

	return result
}

// HasSpecialOrdering returns true if episodes of the given show need reordering
func (m *OfflineOrderingMap) HasSpecialOrdering(showID int64) bool {
	_, ok := m.Table[showID]
	return ok
}

// FromProductionToAired takes production episode number, returns aired episode number
func (m *OfflineOrderingMap) FromProductionToAired(showID int64, season int, episode int) (airedSeason int, airedEpisode int) {
	airedSeason = season
	airedEpisode = episode

	tvShow, ok := m.Table[showID]
	if !ok {
		return
	}

	epList, ok := tvShow[season]
	if !ok {
		return
	}

	if episode >= len(epList) {
		return
	}

	epNum := epList[episode-1]
	n, err := fmt.Sscanf(epNum, "s%de%d", &airedSeason, &airedEpisode)

	if n != 2 || err != nil {
		log.Println(err)
	}

	log.Printf("Mapping s%02de%02d to s%02de%02d\n", season, episode, airedSeason, airedEpisode)
	return
}

func main() {
	shows := LoadConfig("ordering.yaml")
	log.Printf("%+v\n", shows)

	proxy := NewProxyWithScraperAdapters(&TMDBScraperOrderingAdapter{NewOfflineOrderingMap(shows)})
	proxy.Verbose = false

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
