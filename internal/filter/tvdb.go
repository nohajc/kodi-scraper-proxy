package filter

import "io"

// TVDBScraperOrderingAdapter changes ordering of episodes based on provided mapping
type TVDBScraperOrderingAdapter struct {
	OrderingMap EpisodeOrderingMap
}

// AppliesTo returns boolean indicating whether the adapter applies to the given requestHost
func (*TVDBScraperOrderingAdapter) AppliesTo(requestHost string) bool {
	return requestHost == "api.thetvdb.com"
}

// ResponseBodyFilter reads request from in, potentially modifies it and writes the result to out
func (adp *TVDBScraperOrderingAdapter) ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestHost string, requestPath string) {
	// TODO: implement
	defer out.Close()
	io.Copy(out, in)
}
