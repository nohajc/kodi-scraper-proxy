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
	// FromProductionToAired takes episode number in production order
	// and returns the corresponding episode number in aired order
	FromProductionToAired(showID int, season int, episode int) (int, int)
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
	StillPath      interface{}          `json:"still_path"`
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
	if strings.Contains(ctx.Req.URL.Path, "/season/") {
		var tvShowID int
		var seasonNum int
		n, err := fmt.Sscanf(ctx.Req.URL.Path, "/3/tv/%d/season/%d", &tvShowID, &seasonNum)
		if n != 2 || err != nil {
			log.Printf("Error parsing season number from request path: %v\n", err)
			return resp
		}

		log.Printf("Requested TV show %d, season %d\n", tvShowID, seasonNum)
		/*body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Error reading response!")
			return resp
		}*/

		dec := json.NewDecoder(resp.Body)
		dec.UseOrderedObject()
		parsedResponse := ScraperResponse{}
		err = dec.Decode(&parsedResponse)
		if err != nil {
			log.Printf("Error parsing JSON from response: %v\n", err)
			return resp
		}

		//log.Printf("Parsed response: %#v\n", parsedResponse)

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
	}
	return resp
}

// SlidersOrderingMap hardcoded
type SlidersOrderingMap struct{}

// FromProductionToAired takes production episode number, returns aired episode number
func (*SlidersOrderingMap) FromProductionToAired(showID int, season int, episode int) (int, int) {
	if showID != 1649 {
		return season, episode
	}
	switch season {
	case 1:
		mapping := []int{1, 2, 6, 5, 3, 4, 8, 7, 9, 10}
		return season, mapping[episode-1]
	case 2:
		mapping := []int{1, 6, 5, 2, 4, 13, 3, 9, 12, 8, 7, 10, 11}
		return season, mapping[episode-1]
	case 3:
		mapping := []int{2, 1, 10, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 20, 16, 17, 18, 21, 19, 24, 22, 23, 25}
		return season, mapping[episode-1]
	default:
		return season, episode
	}
}

func main() {
	proxy := NewProxyWithScraperAdapters(&TMDBScraperOrderingAdapter{&SlidersOrderingMap{}})
	proxy.Verbose = false

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
