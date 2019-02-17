package main

import (
	"io"
	"log"
	"os"

	"github.com/nohajc/kodi-scraper-proxy/internal/filter"
	"github.com/nohajc/kodi-scraper-proxy/pkg/api"
)

// MuxAdapter multiplexes any number of adapters into one
type MuxAdapter struct {
	Adapters []api.ResponseAdapter
}

// AppliesTo returns true if any of the multiplexed adapters can be applied
func (m *MuxAdapter) AppliesTo(requestHost string) bool {
	for _, adp := range m.Adapters {
		if adp.AppliesTo(requestHost) {
			return true
		}
	}
	return false
}

// ResponseBodyFilter is delegated to the first multiplexed adapter which applies to the given host
func (m *MuxAdapter) ResponseBodyFilter(in io.ReadCloser, out io.WriteCloser, requestHost string, requestPath string) {
	for _, adp := range m.Adapters {
		if adp.AppliesTo(requestHost) {
			adp.ResponseBodyFilter(in, out, requestHost, requestPath)
			return
		}
	}

	defer out.Close()
	io.Copy(out, in)
}

// PluginResponseAdapter exposes functionality of this plugin
var PluginResponseAdapter MuxAdapter

func init() {
	orderingCfgPath := os.Getenv("ORDERING_CONFIG")
	if orderingCfgPath == "" {
		log.Println("Error: ORDERING_CONFIG has to be specified ")
		return
	}

	shows := filter.LoadConfig(orderingCfgPath)
	//log.Printf("%+v\n", shows)
	orderingMap := filter.NewOfflineOrderingMap(shows)

	tmdbAdapter := &filter.TMDBScraperOrderingAdapter{
		OrderingMap: orderingMap,
	}

	tvdbAdapter := &filter.TVDBScraperOrderingAdapter{
		OrderingMap: orderingMap,
	}

	PluginResponseAdapter = MuxAdapter{
		[]api.ResponseAdapter{tmdbAdapter, tvdbAdapter},
	}

	log.Println("Succesfully loaded scraper_filter plugin")
}

func main() {}
