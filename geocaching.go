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
	"sort"
	"strings"
	"time"

	"log"
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

	LastFoundTime time.Time // This is a parsed version of LastFoundDate
}

// This comparitor is used to sort a slice of Geocaches by LastFoundDate.
// LastFoundDate is in the format "2023-01-29T10:08:20"
func (g Geocache) LessFoundDate(other Geocache) bool {
	// Otherwise, compare the dates
	return g.LastFoundTime.Before(other.LastFoundTime)
}

type GeocacheSearchResponse struct {
	Results []Geocache `json:"results"`
	Total   int        `json:"total"`
}

// This runs the query against the geocaching API and returns a slice of up to `take` geocaches,
// and the total number of geocaches matching that query
func (g *GeocachingAPI) searchQuery(lat, long float64, skip, take int) ([]Geocache, int, error) {
	var err error
	req, err := http.NewRequest("GET", "https://www.geocaching.com/api/proxy/web/search/v2", nil)
	if err != nil {
		return nil, 0, err
	}
	query := req.URL.Query()
	query.Add("skip", fmt.Sprint(skip))
	query.Add("take", fmt.Sprint(take))
	query.Add("asc", "true")
	// Note: Sorting by anything other than distance is a "premium feature." This means we
	// have to query all pages of results and sort them ourselves.
	query.Add("sort", "distance")
	query.Add("properties", "callernote")
	query.Add("origin", fmt.Sprintf("%f,%f", lat, long)) //"-27.46794,153.02809"
	query.Add("rad", "16000")
	query.Add("oid", "3356")
	query.Add("ot", "city")
	req.URL.RawQuery = query.Encode()

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
		return nil, 0, err
	}
	defer resp.Body.Close()

	var body []byte
	if body, err = io.ReadAll(resp.Body); err != nil {
		return nil, 0, err
	}

	// Check if the Content-Encoding is gzip, abort if not.
	if resp.Header.Get("Content-Encoding") != "gzip" {
		return nil, 0, fmt.Errorf("unknown Content-Encoding: %s", resp.Header.Get("Content-Encoding"))
	}

	var r io.Reader
	if r, err = gzip.NewReader(bytes.NewReader(body)); err != nil {
		return nil, 0, err
	}
	if body, err = io.ReadAll(r); err != nil {
		return nil, 0, err
	}
	// Unmarshal body into a GeocacheSearchResponse
	var searchResponse GeocacheSearchResponse
	if err = json.Unmarshal(body, &searchResponse); err != nil {
		return nil, 0, err
	}
	// Iterate over the results and parse the LastFoundDate
	for i := 0; i < len(searchResponse.Results); i++ {
		if searchResponse.Results[i].LastFoundDate != "" {
			searchResponse.Results[i].LastFoundTime, _ = time.Parse(time.RFC3339[:19], searchResponse.Results[i].LastFoundDate)
		}
	}

	return searchResponse.Results, searchResponse.Total, nil
}

// This finds all geocaches
func (g *GeocachingAPI) Search(lat, long float64) ([]Geocache, error) {
	var err error
	var results []Geocache
	log.Println("Running a search")

	// Run the first query to get the total number of results
	var total int
	if results, total, err = g.searchQuery(lat, long, 0, 500); err != nil {
		return nil, err
	}

	// Run the rest of the queries to get the rest of the results
	for i := 500; i < total; i += 500 {
		var nextResults []Geocache
		if nextResults, _, err = g.searchQuery(lat, long, i, 500); err != nil {
			return nil, err
		}
		results = append(results, nextResults...)
	}

	// Sort the results using the LessFoundDate comparitor
	sort.Slice(results, func(i, j int) bool {
		return results[i].LessFoundDate(results[j])
	})

	return results, nil
}

// This returns all geocaches with a LastFoundDate later than the given date
func (g *GeocachingAPI) SearchSince(lat, long float64, since time.Time) ([]Geocache, error) {
	var err error
	var results []Geocache

	if results, err = g.Search(lat, long); err != nil {
		return nil, err
	}
	// 2023/02/22 23:23:23 Authenticated
	// 2023/02/22 23:23:23 Running a search
	// 2023/02/22 23:23:26 Before the filter, there are 1702 results
	// 2023/02/22 23:23:26 The first one is  {8952599 Clean Up South Brisbane Cemetery, Dutton Park GCA49PE false 0 13 6 1 2 0 {-27.499583 153.024767} /geocache/GCA49PE false 2023-03-05T09:30:00 {PR6XD7R McLookers} 0001-01-01T00:00:00 0 Queensland Australia [{66 Teamwork cache true} {28 Public restrooms nearby true} {26 Public transportation nearby true} {39 Thorns true}] 2.2mi S 0001-01-01 00:00:00 +0000 UTC}
	// 2023/02/22 23:23:26 The last one is  {34483 Ku-ta views GC86B3 false 64 4 5 1.5 1 0 {-27.4851333333333 152.959033333333} /geocache/GC86B3 false 2002-08-30T00:00:00 {PRG08T tonyjago} 2023-02-22T20:45:52 0 Queensland Australia [{42 Needs maintenance true}] 4.4mi W 2023-02-22 20:45:52 +0000 UTC}
	// 2023/02/22 23:23:26 Found 505 geocaches
	// 2023/02/22 23:23:26 First one is {7819830 Hedge Screen - Rochedale Pictures 04 GC8X8ZG true 2 8 2 4 2 0 {0 0} /geocache/GC8X8ZG false 2020-07-25T00:00:00 {PRC80AY RoddyC} 2023-02-05T09:10:33 0 Queensland Australia [{7 Takes less than one hour true} {25 Parking nearby true} {63 Recommended for tourists true}] 8.4mi SE 2023-02-05 09:10:33 +0000 UTC}
	// 2023/02/22 23:23:26 Last one is {34483 Ku-ta views GC86B3 false 64 4 5 1.5 1 0 {-27.4851333333333 152.959033333333} /geocache/GC86B3 false 2002-08-30T00:00:00 {PRG08T tonyjago} 2023-02-22T20:45:52 0 Queensland Australia [{42 Needs maintenance true}] 4.4mi W 2023-02-22 20:45:52 +0000 UTC}

	// This weirdness is because some geocaches are actually events, and they don't have a LastFoundDate
	// so they end up with 0001-01-01T00:00:00 as their LastFoundDate.

	// The results are sorted by LastFoundDate, so we can just iterate backwards until we find
	// the first result that is before the given date.
	for i := len(results) - 1; i >= 0; i-- {
		if results[i].LastFoundTime.Before(since) {
			return results[i+1:], nil
		}
	}
	return results, nil
}
