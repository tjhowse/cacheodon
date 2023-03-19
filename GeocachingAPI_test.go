package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
)

func TestAuthSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/signin" {
			t.Errorf("Expected to request '/account/signin', got: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`Bloopa doopa this is the body
			of the message
			name="__RequestVerificationToken" type="hidden" value="plooybloots" />
			that was the token you're after.`))
		} else if r.Method == http.MethodPost {
			// Check the body of the POST contains the required fields.
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			// Sheesh this is a lot of boilerplate. Surely there's a shortcut I could use.
			if want, got := "client_id", r.Form.Get("UsernameOrEmail"); want != got {
				t.Errorf("Expected UsernameOrEmail to be '%s', got: %s", want, got)
			}
			if want, got := "client_secret", r.Form.Get("Password"); want != got {
				t.Errorf("Expected Password to be '%s', got: %s", want, got)
			}
			if want, got := "plooybloots", r.Form.Get("__RequestVerificationToken"); want != got {
				t.Errorf("Expected __RequestVerificationToken to be '%s', got: %s", want, got)
			}
			if want, got := "/play", r.Form.Get("ReturnUrl"); want != got {
				t.Errorf("Expected ReturnUrl to be '%s', got: %s", want, got)
			}
			w.Header().Add("Set-Cookie", "gspkauth=verysecretindeed; expires=Fri, 31-Dec-9999 23:59:59 GMT; path=/; secure; HttpOnly")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`"isLoggedIn": true,`))

		}
	}))
	defer server.Close()

	c := APIConfig{
		GeocachingAPIURL: server.URL,
		UnThrottle:       true,
	}

	gc, err := NewGeocachingAPI(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := gc.Auth("client_id", "client_secret"); err != nil {
		t.Fatal(err)
	}
}

func TestAuthFail1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/signin" {
			t.Errorf("Expected to request '/account/signin', got: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`Bloopa doopa this is the body
			of the message
			name="__RequestVerificationToken" type="hidden" value="plooybloots" />
			that was the token you're after.`))
		} else if r.Method == http.MethodPost {
			// Check the body of the POST contains the required fields.
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`It seems your Anti-Forgery Token is invalid`))

		}
	}))
	defer server.Close()

	c := APIConfig{
		GeocachingAPIURL: server.URL,
		UnThrottle:       true,
	}

	gc, err := NewGeocachingAPI(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := gc.Auth("client_id", "client_secret"); err == nil {
		t.Fatal("Should've got an error, but didn't")
	} else {
		if want, got := "Anti-Forgery Token is invalid", err.Error(); want != got {
			t.Errorf("Expected error to be '%s', got: %s", want, got)
		}
	}
}

func TestAuthFail2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/signin" {
			t.Errorf("Expected to request '/account/signin', got: %s", r.URL.Path)
		}
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`Bloopa doopa this is the body
			of the message
			name="__RequestVerificationToken" type="hidden" value="plooybloots" />
			that was the token you're after.`))
		} else if r.Method == http.MethodPost {
			// Check the body of the POST contains the required fields.
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`"isLoggedIn": false,`))

		}
	}))
	defer server.Close()

	c := APIConfig{
		GeocachingAPIURL: server.URL,
		UnThrottle:       true,
	}

	gc, err := NewGeocachingAPI(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := gc.Auth("client_id", "client_secret"); err == nil {
		t.Fatal("Should've got an error, but didn't")
	} else {
		if want, got := "login failed", err.Error(); want != got {
			t.Errorf("Expected error to be '%s', got: %s", want, got)
		}
	}
}

func TestSearchQuery(t *testing.T) {
	searchLat := float32(51.0)
	searchLon := float32(22.0)
	gofakeit.Seed(0)
	// Generate a bunch of fake geocache data using: https://github.com/brianvoe/gofakeit
	// At least 1001 caches to get the pagination working.
	totalCaches := 1001
	fakeCaches := make([]Geocache, totalCaches)
	fakeLogs := make([]GeocacheLog, totalCaches)
	for i := 0; i < totalCaches; i++ {
		gofakeit.Struct(&fakeCaches[i])
		fakeCaches[i].PlacedDate = fakeCaches[i].PlacedDate[:len(fakeCaches[i].PlacedDate)-1]
		fakeCaches[i].LastFoundDate = fakeCaches[i].LastFoundDate[:len(fakeCaches[i].LastFoundDate)-1]
		fakeCaches[i].DetailsURL = "https://www.geocaching.com/geocache/" + fakeCaches[i].Code
		// TODO generate fake logs too.
		gofakeit.Struct(&fakeLogs[i])
		fakeLogs[i].CacheID = fakeCaches[i].ID
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Path == "/account/signin" {
				w.Write([]byte(`Bloopa doopa this is the body
				of the message
				name="__RequestVerificationToken" type="hidden" value="plooybloots" />
				that was the token you're after.`))
			} else if r.URL.Path == "/api/proxy/web/search/v2" {
				// Check the URL contains the correct search terms.
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				originString := fmt.Sprintf("%f,%f", searchLat, searchLon)
				if want, got := originString, r.Form.Get("origin"); want != got {
					t.Errorf("Expected origin to be '%s', got: %s", want, got)
				}
				if want, got := "1234", r.Form.Get("rad"); want != got {
					t.Errorf("Expected radius to be '%s', got: %s", want, got)
				}
				// Parse the skip and take form values to integers
				var skip, take int
				var err error
				if skip, err = strconv.Atoi(r.Form.Get("skip")); err != nil {
					t.Fatal(err)
				}
				if take, err = strconv.Atoi(r.Form.Get("take")); err != nil {
					t.Fatal(err)
				}

				var searchResponse GeocacheSearchResponse
				searchResponse.Total = totalCaches
				searchResponse.Results = fakeCaches[skip:int(math.Min(float64(skip+take), float64(totalCaches)))]

				w.WriteHeader(http.StatusOK)

				// Encode the response as JSON
				enc := json.NewEncoder(w)
				if err := enc.Encode(searchResponse); err != nil {
					t.Fatal(err)
				}
			} else if strings.Split(r.URL.Path, "/")[0] == "geocache" {
				fmt.Println("Got a lookup on a geocache's page.")
			}
		} else if r.Method == http.MethodPost {
			if r.URL.Path == "/account/signin" {
				w.Header().Add("Set-Cookie", "gspkauth=verysecretindeed; expires=Fri, 31-Dec-9999 23:59:59 GMT; path=/; secure; HttpOnly")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`"isLoggedIn": true,`))
			}
		}
	}))
	defer server.Close()
	c := APIConfig{
		GeocachingAPIURL: server.URL,
		UnThrottle:       true,
	}
	st := searchTerms{
		Latitude:      searchLat,
		Longitude:     searchLon,
		RadiusMeters:  1234,
		IgnorePremium: false,
	}
	gc, err := NewGeocachingAPI(c)
	if err != nil {
		t.Fatal(err)
	}
	if caches, err := gc.Search(st); err != nil {
		t.Fatal(err)
	} else {
		if want, got := totalCaches, len(caches); want != got {
			t.Errorf("Expected %d caches, got: %d", want, got)
		}
	}
	st.IgnorePremium = true
	if caches, err := gc.Search(st); err != nil {
		t.Fatal(err)
	} else {
		if want, got := totalCaches, len(caches); want < got {
			t.Errorf("Expected fewer than %d caches, got: %d", want, got)
		}
	}
}
