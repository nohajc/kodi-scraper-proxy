package filter

import (
	"fmt"
	"log"
)

// OfflineOrderingMap contains episode ordering map loaded from an offline resource
type OfflineOrderingMap struct { // TODO: the YAML file should use its own IDs and provide mapping to TMDB/TVDB IDs
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
func (m *OfflineOrderingMap) HasSpecialOrdering(showID int64) bool { // TODO: showID will be a string, e.g. "tmdb-1649" or "tvdb-76557"
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

	if episode < 1 || episode > len(epList) {
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
