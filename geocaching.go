package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type GeocachingAPI struct {
	client    *http.Client
	cookieJar *cookiejar.Jar
	ctx       context.Context
}

func NewGeocachingAPI(ctx context.Context) (*GeocachingAPI, error) {
	var err error
	g := &GeocachingAPI{}
	g.ctx = ctx
	g.cookieJar, err = cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	g.client = &http.Client{
		Jar: g.cookieJar,
	}
	return g, nil
}

// Broadly copying from https://github.com/btittelbach/gctools/blob/master/geocachingsitelib.py and
// https://github.com/cgeo/cgeo/blob/master/main/src/main/java/cgeo/geocaching/connector/gc/GCWebAPI.java
func (g *GeocachingAPI) Auth(clientID, clientSecret string) error {
	var err error

	// First we have to initiate a request to https://www.geocaching.com/account/signin
	// to obtain a "__RequestVerificationToken" value.
	RVTReq, err := http.NewRequest("GET", "https://www.geocaching.com/account/signin", nil)
	if err != nil {
		return err
	}
	RVTResp, err := g.client.Do(RVTReq)
	if err != nil {
		return err
	}
	defer RVTResp.Body.Close()

	RVT := ""

	RVTBody, err := io.ReadAll(RVTResp.Body)
	if err != nil {
		return err
	}

	// These bastards hide the token in a hidden field in the page. There's a cookie by the same name,
	// but it isn't used for authentication, as far as I ca tell.
	rgx := regexp.MustCompile("name=\"__RequestVerificationToken\"\\s+type=\"hidden\"\\s+value=\"([^\"]+)\"")
	matches := rgx.FindStringSubmatch(string(RVTBody))
	if len(matches) < 1 {
		return fmt.Errorf("could not find __RequestVerificationToken")
	}
	RVT = rgx.FindStringSubmatch(string(RVTBody))[1]

	params := url.Values{}
	params.Add("__RequestVerificationToken", RVT)
	params.Add("ReturnUrl", `/play`)
	params.Add("UsernameOrEmail", clientID)
	params.Add("Password", clientSecret)
	body := strings.NewReader(params.Encode())

	POSTReq, err := http.NewRequest("POST", "https://www.geocaching.com/account/signin", body)
	if err != nil {
		return err
	}
	POSTReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/109.0")
	POSTReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	POSTReq.Header.Set("Accept-Language", "en-US,en;q=0.5")
	POSTReq.Header.Set("Accept-Encoding", "gzip, deflate, br")
	POSTReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	POSTReq.Header.Set("Origin", "https://www.geocaching.com")
	POSTReq.Header.Set("Dnt", "1")
	POSTReq.Header.Set("Connection", "keep-alive")
	POSTReq.Header.Set("Referer", "https://www.geocaching.com/account/signin?returnUrl=%2fplay")
	POSTReq.Header.Set("Upgrade-Insecure-Requests", "1")
	POSTReq.Header.Set("Sec-Fetch-Dest", "document")
	POSTReq.Header.Set("Sec-Fetch-Mode", "navigate")
	POSTReq.Header.Set("Sec-Fetch-Site", "same-origin")
	POSTReq.Header.Set("Sec-Fetch-User", "?1")

	POSTResp, err := g.client.Do(POSTReq)
	if err != nil {
		return err
	}
	defer POSTResp.Body.Close()
	if body, err := io.ReadAll(POSTResp.Body); err == nil {
		if match, err := regexp.Match("It seems your Anti-Forgery Token is invalid", body); err == nil && match {
			return fmt.Errorf("Anti-Forgery Token is invalid")
		}
		if match, err := regexp.Match(`"isLoggedIn": true,`, body); err != nil || !match {
			return fmt.Errorf("login failed")
		}
	} else {
		return fmt.Errorf("couldn't read body")
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
	Difficulty        float64 `json:"difficulty"`
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

type GeocacheSearchResponse struct {
	Results []Geocache `json:"results"`
	Total   int        `json:"total"`
}

func (g *GeocachingAPI) Search(lat, long float64) ([]Geocache, error) {
	var err error
	fmt.Println("Running a search")

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

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, err
	}

	// Check if the Content-Encoding is gzip, and unzip it if so
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var r io.Reader
		if r, err = gzip.NewReader(bytes.NewReader(body)); err != nil {
			return nil, err
		}
		if body, err = io.ReadAll(r); err != nil {
			return nil, err
		}
		// Unmarshal body into a GeocacheSearchResponse
		var searchResponse GeocacheSearchResponse
		if err = json.Unmarshal(body, &searchResponse); err != nil {
			return nil, err
		}
		fmt.Println("Found", searchResponse.Total, "geocaches")

		// TODO Repeat the request with different skip and take values until all geocaches are found

		return searchResponse.Results, nil
	}

	return nil, nil
}
