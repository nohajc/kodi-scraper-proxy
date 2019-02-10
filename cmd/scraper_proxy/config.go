package main

import (
	"log"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// EpisodeMap is the episode mapping of each season of a show
type EpisodeMap struct {
	Season   int      `yaml:"season"`
	Episodes []string `yaml:"episodes"`
}

// TVShow contains show's name (optional), its TMDB ID and an array of EpisodMaps
type TVShow struct {
	Name     string       `yaml:"name"`
	ID       int64        `yaml:"id"`
	Ordering []EpisodeMap `yaml:"ordering"`
}

// Config contains an array of TVShows
type Config struct {
	TVShows []TVShow `yaml:"tv-shows"`
}

// LoadConfig from the given yaml file
func LoadConfig(filePath string) Config {
	var cfg Config

	file, err := os.Open(filePath)
	if err != nil {
		log.Println(err)
		return Config{}
	}

	d := yaml.NewDecoder(file)
	err = d.Decode(&cfg)
	if err != nil {
		log.Println(err)
		return Config{}
	}

	return cfg
}
