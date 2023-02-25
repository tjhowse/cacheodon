package main

import (
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type configStore struct {
	// TODO migrate this to use koanf:
	// https://github.com/knadh/koanf

	// TODO add a table that tracks the number of times a person finds a cache in a day
	// so we can add text to the end of the message like "That's their 3rd find today!"
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
