package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type GeocachingAPI struct {
	c   *http.Client
	ctx context.Context
}

func NewGeocachingAPI(ctx context.Context) *GeocachingAPI {
	g := &GeocachingAPI{}
	g.ctx = ctx
	return g
}

// Broadly copying from https://github.com/btittelbach/gctools/blob/master/geocachingsitelib.py and
// https://github.com/cgeo/cgeo/blob/master/main/src/main/java/cgeo/geocaching/connector/gc/GCWebAPI.java

func (g *GeocachingAPI) Auth(clientID, clientSecret string) error {
	// Set up a cookie jar so we can store cookies between requests.
	var err error
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	// First we have to initiate a request to https://www.geocaching.com/account/signin
	// to obtain a "__RequestVerificationToken" cookie.
	req, err := http.NewRequest("GET", "https://www.geocaching.com/account/signin", nil)
	if err != nil {
		return err
	}
	g.c = &http.Client{
		Jar: jar,
	}
	resp, err := g.c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	RVT := ""
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "__RequestVerificationToken" {
			RVT = cookie.Value
		}
	}
	if RVT == "" {
		return fmt.Errorf("could not find __RequestVerificationToken")
	}

	// Print out all the cookies in our cookie jar for checking
	fmt.Println("All cookies:")
	for _, cookie := range jar.Cookies(req.URL) {
		fmt.Println(cookie.Name, cookie.Value)
	}

	// We can make a POST request to https://www.geocaching.com/account/signin containing
	// __RequestVerificationToken, UsernameOrEmail, Password and ReturnUrl.
	params := url.Values{}
	params.Add("__RequestVerificationToken", RVT)
	params.Add("ReturnUrl", `/play`)
	params.Add("UsernameOrEmail", clientID)
	params.Add("Password", clientSecret)
	body := strings.NewReader(params.Encode())

	req, err = http.NewRequest("POST", "https://www.geocaching.com/account/signin", body)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.geocaching.com")
	req.Header.Set("Dnt", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://www.geocaching.com/account/signin?returnUrl=%2fplay")
	req.Header.Set("Cookie", fmt.Sprintf("__RequestVerificationToken=%s", RVT))
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	resp, err = g.c.Do(req)
	if err != nil {
		return err
	}
	for _, cookie := range resp.Cookies() {
		fmt.Println(cookie.Name, cookie.Value)
	}
	// Print the body of the response
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		fmt.Println(string(body))
	} else {
		return err
	}

	return nil
}

type Geocache struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	PremiumOnly       bool    `json:"premiumOnly"`
	FavoritePoints    int     `json:"favoritePoints"`
	GeocacheType      int     `json:"geocacheType"`
	ContainerType     int     `json:"containerType"`
	Difficulty        int     `json:"difficulty"`
	Terrain           float64 `json:"terrain"`
	CacheStatus       int     `json:"cacheStatus"`
	PostedCoordinates struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"postedCoordinates"`
	DetailsURL string `json:"detailsUrl"`
	HasGeotour bool   `json:"hasGeotour"`
	PlacedDate string `json:"placedDate"`
	Owner      struct {
		Code     string `json:"code"`
		Username string `json:"username"`
	} `json:"owner"`
	LastFoundDate  string `json:"lastFoundDate"`
	TrackableCount int    `json:"trackableCount"`
	Region         string `json:"region"`
	Country        string `json:"country"`
	Attributes     []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		IsApplicable bool   `json:"isApplicable"`
	} `json:"attributes"`
	Distance string `json:"distance"`
	Bearing  string `json:"bearing"`
}

func (g *GeocachingAPI) Search(lat, long float64) ([]Geocache, error) {

	req, err := http.NewRequest("GET", "https://www.geocaching.com/api/proxy/web/search/v2?skip=0&take=500&asc=true&sort=distance&properties=callernote&origin=-27.46794%2C153.02809&rad=16000&oid=3356&ot=city", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.geocaching.com/play")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", "BMItemsPerPage=1000;-H Sec-Fetch-Dest:")

	resp, err := g.c.Do(req)
	if err != nil {
		return nil, err
	}
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		fmt.Println(string(body))
	} else {
		return nil, err
	}

	defer resp.Body.Close()
	return nil, nil
}
