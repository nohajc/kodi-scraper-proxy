package filter

import (
	"fmt"
	"io"
	"log"
	"strings"

	json "github.com/virtuald/go-ordered-json"
)

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

// TMDBScraperOrderingAdapter changes ordering of episodes based on provided mapping
type TMDBScraperOrderingAdapter struct {
	OrderingMap EpisodeOrderingMap
}

// AppliesTo returns boolean indicating whether the adapter applies to the given requestHost
func (*TMDBScraperOrderingAdapter) AppliesTo(requestHost string) bool {
	return requestHost == "api.themoviedb.org"
}

// ResponseBodyFilter reads request from in, potentially modifies it and writes the result to out
func (adp *TMDBScraperOrderingAdapter) ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestHost string, requestPath string) {
	if !adp.responseBodyFilterInternal(in, out, requestPath) {
		go func() {
			defer out.Close()
			io.Copy(out, in)
		}()
	}
}

// responseBodyFilterInternal takes response body Reader (in) and indicates if it is going to modify the response or not
func (adp *TMDBScraperOrderingAdapter) responseBodyFilterInternal(in io.ReadCloser, out io.WriteCloser, requestURL string) bool {
	if !strings.Contains(requestURL, "/season/") {
		return false
	}

	var tvShowID int64
	var seasonNum int
	n, err := fmt.Sscanf(requestURL, "/3/tv/%d/season/%d", &tvShowID, &seasonNum)
	if n != 2 || err != nil {
		log.Printf("Error parsing season number from request path: %v\n", err)
		return false
	}

	log.Printf("Requested TV show %d, season %d\n", tvShowID, seasonNum)

	if !adp.OrderingMap.HasSpecialOrdering(tvShowID) {
		log.Println("Doesn't need reordering")
		return false
	}

	log.Println("Needs reordering")

	go func() {
		defer out.Close()

		dec := json.NewDecoder(in)
		dec.UseOrderedObject()
		parsedResponse := ScraperResponse{}
		err = dec.Decode(&parsedResponse)
		if err != nil {
			log.Printf("Error parsing JSON from response: %v\n", err)
			return
		}

		episodesReordered := make([]Episode, len(parsedResponse.Episodes))
		for i := range parsedResponse.Episodes {
			_, airedEpNum := adp.OrderingMap.FromProductionToAired(tvShowID, seasonNum, i+1)
			episodesReordered[i] = parsedResponse.Episodes[airedEpNum-1]
			episodesReordered[i].EpisodeNumber = i + 1
		}
		parsedResponse.Episodes = episodesReordered

		enc := json.NewEncoder(out)
		err = enc.Encode(parsedResponse)
		if err != nil {
			log.Printf("Error serializing modified response: %v\n", err)
		}
	}()
	return true
}
