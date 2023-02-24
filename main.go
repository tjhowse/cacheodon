package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type configStore struct {
	LastUpdateTime time.Time
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
	return ioutil.WriteFile(c.Filename, b, 0644)
}

// Load the current config from a toml file.
func (c *config) Load() error {
	b, err := ioutil.ReadFile(c.Filename)
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

	ctx := context.Background()
	g, _ := NewGeocachingAPI(ctx)
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
		// TODO This should read the coordinates from the config store.
		if searchResults, err = g.SearchSince(-27.46794, 153.02809, config.Store.LastUpdateTime); err != nil {
			log.Fatal(err)
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
				config.Store.LastUpdateTime = gc.LastFoundTime
				if err := config.Save(); err != nil {
					log.Fatal(err)
					os.Exit(1)
				}
			}

		}
		time.Sleep(1 * time.Minute)
	}

}
