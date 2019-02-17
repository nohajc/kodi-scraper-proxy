package filter

// EpisodeOrderingMap is an interface for mapping episode numbers to different numbers
type EpisodeOrderingMap interface {
	// HasSpecialOrdering returns true if episodes of the given show need reordering
	HasSpecialOrdering(showID string) bool
	// FromProductionToAired takes episode number in production order
	// and returns the corresponding episode number in aired order
	FromProductionToAired(showID string, season int, episode int) (int, int)
	// ProductionEpisodeCount returns number of episodes in the given production season
	// (only available if there is a reordering map for the season, returns 0 otherwise)
	ProductionEpisodeCount(showID string, season int) int
}
