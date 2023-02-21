package main

import (
	"context"
	"log"
	"os"

	"github.com/mattn/go-mastodon"
)

type Mastodon struct {
	c *mastodon.Client
}

func (m *Mastodon) PostStatus(status string) error {
	_, err := m.c.PostStatus(context.Background(), &mastodon.Toot{
		Status: status,
	})
	return err
}

func NewMastodon() *Mastodon {
	m := &Mastodon{}
	m.c = mastodon.NewClient(&mastodon.Config{
		Server:       os.Getenv("MASTODON_SERVER"),
		ClientID:     os.Getenv("MASTODON_CLIENT_ID"),
		ClientSecret: os.Getenv("MASTODON_CLIENT_SECRET"),
	})
	err := m.c.Authenticate(context.Background(), os.Getenv("MASTODON_USER_EMAIL"), os.Getenv("MASTODON_USER_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}
	return m
}
