package filter

import (
	"io"
	"io/ioutil"
	"log"
)

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
	defer out.Close()
	if requestPath == "/login" {
		token, _ := ioutil.ReadAll(in)
		log.Printf("JWT token: %s", string(token))
		_, _ = out.Write(token)
	} else {
		// TODO: implement
		io.Copy(out, in)
	}
}
