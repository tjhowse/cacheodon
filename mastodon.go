package main

import (
	"context"
	"os"

	"github.com/mattn/go-mastodon"
)

type Mastodon struct {
	c *mastodon.Client
}

func NewMastodon() (*Mastodon, error) {
	m := &Mastodon{}
	m.c = mastodon.NewClient(&mastodon.Config{
		Server:       os.Getenv("MASTODON_SERVER"),
		ClientID:     os.Getenv("MASTODON_CLIENT_ID"),
		ClientSecret: os.Getenv("MASTODON_CLIENT_SECRET"),
	})
	err := m.c.Authenticate(context.Background(), os.Getenv("MASTODON_USER_EMAIL"), os.Getenv("MASTODON_USER_PASSWORD"))
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Posts a status update
func (m *Mastodon) PostStatus(status string) error {
	_, err := m.c.PostStatus(context.Background(), &mastodon.Toot{
		Status: status,
	})
	return err
}

// Gets my last `n` statuses
func (m *Mastodon) GetMyStatuses(n int64) ([]*mastodon.Status, error) {
	if account, err := m.c.GetAccountCurrentUser(context.Background()); err != nil {
		return nil, err
	} else {
		return m.c.GetAccountStatuses(context.Background(), account.ID, &mastodon.Pagination{
			Limit: n,
		})
	}
}
