package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

type mockGeocachingApi struct {
	caches []Geocache
	logs   []GeocacheLog
}

// Populate some dummy data into the struct
func (m *mockGeocachingApi) populate() {
	m.caches = []Geocache{
		{
			ID:             123456,
			Name:           "Secret Hideout",
			Code:           "GC1234",
			PremiumOnly:    true,
			FavoritePoints: 5,
			GeocacheType:   1,
			ContainerType:  2,
			Difficulty:     3.5,
			Terrain:        2.5,
			CacheStatus:    1,
			PostedCoordinates: struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			}{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
			DetailsURL: "/geocache/GC1234",
			HasGeotour: false,
			PlacedDate: "2020-01-01T00:00:00",
			Owner: struct {
				Code     string `json:"code"`
				Username string `json:"username"`
			}{
				Code:     "ABC123",
				Username: "johndoe",
			},
			LastFoundDate:  "2022-05-01T12:00:00",
			TrackableCount: 2,
			Region:         "California",
			Country:        "United States",
			Attributes: []struct {
				ID           int    `json:"id"`
				Name         string `json:"name"`
				IsApplicable bool   `json:"isApplicable"`
			}{
				{ID: 24, Name: "Wheelchair accessible", IsApplicable: false},
				{ID: 8, Name: "Scenic view", IsApplicable: true},
			},
			Distance:      "2.3mi",
			Bearing:       "NW",
			LastFoundTime: time.Now(),
			GUID:          "a8cf16ab-5a5d-42a2-9a8e-2b33d431c758",
		},
	}
	m.logs = []GeocacheLog{
		{
			LogID:               2150129950,
			CacheID:             123456,
			LogGUID:             "fc59d67c-ccda-45a7-ad5c-f9a09f040d60",
			Latitude:            37.7749,
			Longitude:           -122.4194,
			LatLonString:        "37.7749,-122.4194",
			LogTypeID:           1,
			LogType:             "Attended",
			LogTypeImage:        "1.png",
			LogText:             "Had a great time at this event. Thanks for hosting!\n",
			Created:             "3/16/2023",
			Visited:             "3/16/2023",
			UserName:            "Amy",
			MembershipLevel:     3,
			AccountID:           16998345,
			AccountGUID:         "8fa1b05c-d5f5-4f9c-8f0b-634b88017772",
			Email:               "amy@example.com",
			AvatarImage:         "https://www.example.com/avatar.jpg",
			GeocacheFindCount:   1123,
			GeocacheHideCount:   5,
			ChallengesCompleted: 2,
			IsEncoded:           true,
			Creator: struct {
				GroupTitle    string `json:"GroupTitle"`
				GroupImageURL string `json:"GroupImageUrl"`
			}{
				GroupTitle:    "Premium Member",
				GroupImageURL: "/images/icons/prem_user.gif",
			},
			Images: []any{},
		},
	}
}

// Advance the last found date on the stored cache
func (m *mockGeocachingApi) advanceLastFoundDate() {
	m.caches[0].LastFoundTime = m.caches[0].LastFoundTime.Add(time.Hour * 24)
}

func (m *mockGeocachingApi) Auth(clientID, clientSecret string) error {
	return nil
}

func (m *mockGeocachingApi) Search(st searchTerms) ([]Geocache, error) {
	return m.caches, nil
}

func (m *mockGeocachingApi) GetLogs(geocache *Geocache) ([]GeocacheLog, error) {
	return m.logs, nil
}

func TestUpdate(t *testing.T) {
	var err error
	tempdir := t.TempDir()
	conf := configStore{
		SearchTerms: searchTerms{
			AreaName: "Blerpville",
		},
		DBFilename: tempdir + "/test.sqlite3",
	}
	var g *Geocaching
	api := &mockGeocachingApi{}
	api.populate()
	if g, err = NewGeocaching(conf, api); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer g.Close()
	var logs []postDetails
	logs, err = g.Update()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}

	// TODO check that we get a "New cache appeared!" log message
	for _, log := range logs {
		fmt.Println(log)
	}

	// Advance the last found date on the mock cache
	api.advanceLastFoundDate()
	logs, err = g.Update()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	for _, log := range logs {
		fmt.Println(log)
	}
	// TODO Check that we get the "That's their second find for the day!" thing.
	api.advanceLastFoundDate()
	logs, err = g.Update()
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	for _, log := range logs {
		fmt.Println(log)
	}

}
