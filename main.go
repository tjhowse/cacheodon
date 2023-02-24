package main

import (
	"log"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type configStore struct {
	State struct {
		LastPostedFoundTime time.Time
	}
	SearchTerms struct {
		Latitude     float32
		Longitude    float32
		RadiusMeters int
	}
}

type config struct {
	Filename string
	Store    configStore
}

// Write the current config out to a toml file.
func (c *config) Save() error {
	b, err := toml.Marshal(c.Store)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Filename, b, 0644)
}

// Load the current config from a toml file.
func (c *config) Load() error {
	b, err := os.ReadFile(c.Filename)
	if err != nil {
		return err
	}
	return toml.Unmarshal(b, &c.Store)
}

func NewConfig(filename string) (*config, error) {
	c := &config{
		Filename: filename,
	}
	if err := c.Load(); err != nil {
		if os.IsNotExist(err) {
			c.Store.State.LastPostedFoundTime = time.Now()
			if err := c.Save(); err != nil {
				return nil, err
			}
		}
	}
	return c, nil
}
func main() {
	var err error

	config, err := NewConfig("config.toml")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	g, _ := NewGeocachingAPI("https://www.geocaching.com")
	if err := g.Auth(os.Getenv("GEOCACHING_CLIENT_ID"), os.Getenv("GEOCACHING_CLIENT_SECRET")); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Println("Authenticated")
	var searchResults []Geocache
	m, err := NewMastodon()
	if err != nil {
		log.Println(err)
	}

	for {
		if searchResults, err = g.SearchSince(
			float64(config.Store.SearchTerms.Latitude),
			float64(config.Store.SearchTerms.Longitude),
			config.Store.State.LastPostedFoundTime); err != nil {

			log.Println(err)
			time.Sleep(1 * time.Minute)
			continue
		}
		for _, gc := range searchResults {
			logs, err := g.GetLogs(&gc)
			if err != nil {
				log.Println(err)
				continue
			}
			message := ""
			if len(logs) > 0 {
				message += "\"" + logs[0].UserName + "\""
			} else {
				message += "Someone"
			}
			message += " just found the \"" + gc.Name + "\" geocache! https://www.geocaching.com" + gc.DetailsURL

			if len(logs) > 0 {
				message += " They wrote: \"" + logs[0].LogText + "\""
			}

			log.Println(message)
			if m == nil {
				m, err = NewMastodon()
				if err != nil {
					log.Println(err)
				}
			}
			if len(message) >= 500 {
				message = message[:500]
			}
			if err := m.PostStatus(message); err != nil {
				log.Println(err)
				m = nil
			} else {
				log.Println("Posted to Mastodon: " + message)
				config.Store.State.LastPostedFoundTime = gc.LastFoundTime
				if err := config.Save(); err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
			}

		}
		time.Sleep(1 * time.Minute)
	}

}
