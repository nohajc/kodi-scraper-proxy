package main

import (
	"log"
	"os"

	"github.com/nohajc/kodi-scraper-proxy/internal/filter"
)

// PluginResponseAdapter exposes functionality of this plugin
var PluginResponseAdapter filter.TMDBScraperOrderingAdapter

func init() {
	orderingCfgPath := os.Getenv("ORDERING_CONFIG")
	if orderingCfgPath == "" {
		log.Println("Error: ORDERING_CONFIG has to be specified ")
		return
	}

	shows := filter.LoadConfig(orderingCfgPath)
	//log.Printf("%+v\n", shows)
	PluginResponseAdapter = filter.TMDBScraperOrderingAdapter{
		OrderingMap: filter.NewOfflineOrderingMap(shows),
	}
	log.Println("Succesfully loaded scraper_filter plugin")
}

func main() {}
