package main

import (
	"context"
	"fmt"
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

	for {
		// TODO this should compare to found date of the last geocache we published,
		// not the last time we checked.
		if searchResults, err = g.SearchSince(-27.46794, 153.02809, config.Store.LastUpdateTime); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		// config.Store.LastUpdateTime = time.Now()
		// if err := config.Save(); err != nil {
		// 	log.Fatal(err)
		// 	os.Exit(1)
		// }

		for _, gc := range searchResults {
			log.Println(gc.Name, "[", gc.Code, "] was updated at", gc.LastFoundDate, ". Its PremiumOnly is", gc.PremiumOnly)
		}
		logs, err := g.GetLogs(&searchResults[0])
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}

		for _, log := range logs {
			fmt.Println(log)
		}
		time.Sleep(1 * time.Minute)
	}

	// log.Println("Found", len(searchResults), "geocaches")
	// log.Println("First one is", searchResults[0])
	// log.Println("Last one is", searchResults[len(searchResults)-1])
	// m := NewMastodon()
	// if err := m.PostStatus(fmt.Sprintf("I know about %d geocaches around Brisbane! I'm being taught how to tell you about them in a useful way. Stand by!", len(searchResults))); err != nil {
	// 	log.Fatal(err)
	// }
}
