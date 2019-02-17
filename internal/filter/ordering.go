package filter

import (
	"fmt"
	"log"
)

// OfflineOrderingMap contains episode ordering map loaded from an offline resource
type OfflineOrderingMap struct {
	Table     map[int64]map[int][]string
	ResolveID map[string]int64
}

// NewOfflineOrderingMap creates new instance from Config
func NewOfflineOrderingMap(cfg Config) *OfflineOrderingMap {
	result := &OfflineOrderingMap{
		Table:     make(map[int64]map[int][]string),
		ResolveID: make(map[string]int64),
	}

	for _, tvShow := range cfg.TVShows {
		result.Table[tvShow.ID] = make(map[int][]string)
		for _, epSeasonMap := range tvShow.Ordering {
			result.Table[tvShow.ID][epSeasonMap.Season] = make([]string, len(epSeasonMap.Episodes))
			copy(result.Table[tvShow.ID][epSeasonMap.Season], epSeasonMap.Episodes)
		}
	}

	for _, alias := range cfg.Aliases {
		id := alias.TMDB
		result.ResolveID[fmt.Sprintf("tmdb-%v", id)] = id
		result.ResolveID[fmt.Sprintf("tvdb-%v", alias.TVDB)] = id
	}

	return result
}

// HasSpecialOrdering returns true if episodes of the given show need reordering
func (m *OfflineOrderingMap) HasSpecialOrdering(showID string) bool { // TODO: showID will be a string, e.g. "tmdb-1649" or "tvdb-76557"
	realID, ok := m.ResolveID[showID]
	if ok {
		_, ok = m.Table[realID]
	}
	return ok
}

// FromProductionToAired takes production episode number, returns aired episode number
func (m *OfflineOrderingMap) FromProductionToAired(showID string, season int, episode int) (airedSeason int, airedEpisode int) {
	airedSeason = season
	airedEpisode = episode

	realID, ok := m.ResolveID[showID]
	if !ok {
		return
	}

	tvShow, ok := m.Table[realID]
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

// ProductionEpisodeCount returns number of episodes in a season
func (m *OfflineOrderingMap) ProductionEpisodeCount(showID string, season int) (count int) {
	count = 0

	realID, ok := m.ResolveID[showID]
	if !ok {
		return
	}

	tvShow, ok := m.Table[realID]
	if !ok {
		return
	}

	epList, ok := tvShow[season]
	if !ok {
		return
	}

	count = len(epList)
	return
}
