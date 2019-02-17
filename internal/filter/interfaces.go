package filter

// EpisodeOrderingMap is an interface for mapping episode numbers to different numbers
type EpisodeOrderingMap interface {
	// HasSpecialOrdering returns true if episodes of the given show need reordering
	HasSpecialOrdering(showID int64) bool
	// FromProductionToAired takes episode number in production order
	// and returns the corresponding episode number in aired order
	FromProductionToAired(showID int64, season int, episode int) (int, int)
}
