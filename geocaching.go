package main

import (
	"context"
	"fmt"
	"io/ioutil"
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
	// Set up a cookie jar so we can store cookies between requests.
	var err error

	// First we have to initiate a request to https://www.geocaching.com/account/signin
	// to obtain a "__RequestVerificationToken" cookie.
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

	RVTBody, err := ioutil.ReadAll(RVTResp.Body)
	if err != nil {
		return err
	}

	// Construct a regex with the pattern "name=\"__RequestVerificationToken\"\\s+type=\"hidden\"\\s+value=\"([^\"]+)\""
	// and use it to extract the value of the __RequestVerificationToken from the response body.
	rgx := regexp.MustCompile("name=\"__RequestVerificationToken\"\\s+type=\"hidden\"\\s+value=\"([^\"]+)\"")
	matches := rgx.FindStringSubmatch(string(RVTBody))
	if len(matches) < 1 {
		return fmt.Errorf("could not find __RequestVerificationToken")
	}
	RVT = rgx.FindStringSubmatch(string(RVTBody))[1]

	if RVT == "" {
		return fmt.Errorf("could not find __RequestVerificationToken")
	}
	// We can make a POST request to https://www.geocaching.com/account/signin containing
	// __RequestVerificationToken, UsernameOrEmail, Password and ReturnUrl.
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
	// POSTReq.Header.Set("X-Verification-Token", RVT)
	POSTReq.Header.Set("Upgrade-Insecure-Requests", "1")
	POSTReq.Header.Set("Sec-Fetch-Dest", "document")
	POSTReq.Header.Set("Sec-Fetch-Mode", "navigate")
	POSTReq.Header.Set("Sec-Fetch-Site", "same-origin")
	POSTReq.Header.Set("Sec-Fetch-User", "?1")

	POSTResp, err := g.client.Do(POSTReq)
	if err != nil {
		fmt.Println("Request failed")
		return err
	}
	defer POSTResp.Body.Close()
	// Print the body of the response
	if body, err := ioutil.ReadAll(POSTResp.Body); err == nil {
		if match, err := regexp.Match("It seems your Anti-Forgery Token is invalid", body); err == nil {
			if match {
				return fmt.Errorf("Anti-Forgery Token is invalid")
			}
		} else {
			return err
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
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		fmt.Println(string(body))
	} else {
		return nil, err
	}

	defer resp.Body.Close()
	return nil, nil
}
