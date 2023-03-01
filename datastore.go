package main

import (
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type searchTerms struct {
	Latitude     float32
	Longitude    float32
	RadiusMeters int
	AreaName     string
}

type URLConfig struct {
	// The URL of the Geocaching API.
	GeocachingAPIURL string
	SOCKS5ProxyURL   string
}

type configStore struct {
	State struct {
		LastPostedFoundTime time.Time
	}
	Configuration URLConfig
	SearchTerms   searchTerms
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

func NewDatastore(filename string) (*config, error) {
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
