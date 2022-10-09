package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type postData struct {
	key   string
	value string
}

var theTestData = []struct {
	name   string
	url    string
	method string
	params []postData
	status int
}{
	{"home-page", "/", "GET", []postData{}, http.StatusOK},
	{"login", "/login", "GET", []postData{}, http.StatusOK},
	{"logout", "/logout", "GET", []postData{}, http.StatusOK},
}

func TestHandlers(t *testing.T) {
	routes := getRoutes()
	ts := httptest.NewTLSServer(routes)
	defer ts.Close()

	for _, e := range theTestData {
		if e.method == "GET" {
			resp, err := ts.Client().Get(ts.URL + e.url)

			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != e.status {
				t.Errorf("Expected status code is %d, but get %d", e.status, resp.StatusCode)
			}
		} else {
			values := url.Values{}

			for _, x := range e.params {
				values.Add(x.key, x.value)
			}

			resp, err := ts.Client().PostForm(ts.URL+e.url, values)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != e.status {
				t.Errorf("Expected status code is %d, but get %d", e.status, resp.StatusCode)
			}
		}
	}
}
