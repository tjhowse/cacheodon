package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

	c := Config{
		GeocachingAPIURL: server.URL,
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

	c := Config{
		GeocachingAPIURL: server.URL,
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

	c := Config{
		GeocachingAPIURL: server.URL,
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
