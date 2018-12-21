/*
 *  Copyright (c) 2017-2018 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

package api

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/SamsungSLAV/boruta"
	util "github.com/SamsungSLAV/boruta/http"
	"github.com/SamsungSLAV/boruta/matcher"
	"github.com/SamsungSLAV/boruta/requests"
	"github.com/SamsungSLAV/boruta/workers"
	"github.com/dimfeld/httptreemux"
	"github.com/stretchr/testify/assert"
)

var update bool

const (
	// client
	CORSRequestOrigin  = "Origin"
	CORSRequestMethod  = "Access-Control-Request-Method"
	CORSRequestHeaders = "Access-Control-Request-Headers"

	// server
	CORSAllowOrigin  = "Access-Control-Allow-Origin"
	CORSAllowMethods = "Access-Control-Allow-Methods"
	CORSAllowHeaders = "Access-Control-Allow-Headers"
	CORSMaxAge       = "Access-Control-Max-Age"
)

func TestMain(m *testing.M) {
	flag.BoolVar(&update, "update", false, "update testdata")
	flag.Parse()
	os.Exit(m.Run())
}

func initTest(t *testing.T, origins []string, age int) (*assert.Assertions, *API) {
	wm := workers.NewWorkerList()
	return assert.New(t), NewAPI(requests.NewRequestQueue(wm, matcher.NewJobsManager(wm)), wm,
		origins, age)
}

func prepareHeaders(origin, header, method string, server bool) (hdr http.Header) {
	var originKey, methodKey, headerKey string
	if server {
		originKey = CORSAllowOrigin
		headerKey = CORSAllowHeaders
		methodKey = CORSAllowMethods
	} else {
		originKey = CORSRequestOrigin
		headerKey = CORSRequestHeaders
		methodKey = CORSRequestMethod
	}
	hdr = make(http.Header)
	if origin != "" {
		hdr.Set(originKey, origin)
		if server {
			hdr.Set("Vary", "Origin")
		}
	}
	if header != "" {
		hdr.Set(headerKey, header)
	}
	if method != "" {
		hdr.Set(methodKey, method)
	}
	return
}

func TestPanicHandler(t *testing.T) {
	assert, api := initTest(t, nil, 0)
	msg := "Test PanicHandler: server panic"
	otherErr := "some error"
	tests := [...]struct {
		name string
		path string
		err  interface{}
	}{
		{
			name: "panic-server-error",
			path: "/priv/api/panic/srvErr/",
			err: &util.ServerError{
				Err:    msg,
				Status: http.StatusInternalServerError,
			},
		},
		{
			name: "panic-other-error",
			path: "/priv/api/panic/otherErr/",
			err:  otherErr,
		},
	}
	contentType := "text/plain; charset=utf-8"
	grp := api.r.NewGroup("/")

	newHandler := func(err interface{}) httptreemux.HandlerFunc {
		return func(_ http.ResponseWriter, _ *http.Request, _ map[string]string) {
			panic(err)
		}
	}
	for _, test := range tests {
		grp.GET(test.path, newHandler(test.err))
	}
	srv := httptest.NewServer(api.Router)
	assert.NotNil(srv)
	defer srv.Close()
	for _, test := range tests {
		resp, err := http.Get(srv.URL + test.path)
		assert.Nil(err)
		tdata := filepath.Join("testdata", test.name+".txt")
		body, err := ioutil.ReadAll(resp.Body)
		assert.Nil(err)
		if update {
			ioutil.WriteFile(tdata, body, 0644)
		}
		expected, err := ioutil.ReadFile(tdata)
		assert.Nil(err)
		assert.Equal(http.StatusInternalServerError, resp.StatusCode)
		assert.Equal(contentType, resp.Header.Get("Content-Type"))
		assert.Equal(expected, body)
	}
}

func TestNewAPI(t *testing.T) {
	assert, api := initTest(t, nil, 0)
	assert.NotNil(api)
	api = nil
	assert, api = initTest(t, []string{"*"}, 5)
	assert.NotNil(api)
}

func TestRedirectToDefault(t *testing.T) {
	assert, api := initTest(t, nil, 0)
	srv := httptest.NewServer(api.Router)
	defer srv.Close()

	reqPath := "/api/reqs"
	redirPath := "/api/" + defaultAPI + "/reqs/"
	var i int
	redirCheck := func(req *http.Request, via []*http.Request) error {
		switch {
		case i == 0:
			// Check if proper status code was set.
			assert.Equal(http.StatusPermanentRedirect, req.Response.StatusCode)
			// Check if method hasn't changed.
			assert.Equal(via[0].Method, req.Method, "first redirection")
		case i == 1:
			// Check if proper URL was set.
			assert.Equal(srv.URL+redirPath, req.URL.String())
			// Check if method hasn't changed.
			assert.Equal(via[0].Method, req.Method, "second redirection")
			// It isn't our business if default API does more redirects, but
			// return error when there's more than 10 redirects (as Go does).
		case i > 9:
			return errors.New("too many redirects")
		}
		i++
		return nil
	}

	client := http.Client{
		CheckRedirect: redirCheck,
	}

	// Response is checked in redirCheck.
	_, err := client.Post(srv.URL+reqPath, "text/plain", nil)
	assert.Nil(err)

	// Path to default version of API, which wasn't found.
	badPath := "/api/" + defaultAPI + "/missing/test"
	var srvErr util.ServerError
	resp, err := client.Post(srv.URL+badPath, "text/plain", nil)
	assert.Nil(err)
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	assert.Equal("application/json", resp.Header.Get("Content-Type"))
	err = decoder.Decode(&srvErr)
	assert.Equal(http.StatusNotFound, resp.StatusCode)
	assert.Equal(boruta.NotFoundError(badPath).Error(), srvErr.Err)
}

func TestCORS(t *testing.T) {
	path := "/api/v1/reqs"
	wm := workers.NewWorkerList()
	mr := matcher.NewJobsManager(wm)
	rq := requests.NewRequestQueue(wm, mr)
	srvAll := httptest.NewServer(NewAPI(rq, wm, []string{"*"}, 0).Router)
	srvFoo := httptest.NewServer(NewAPI(rq, wm, []string{"http://foo.bar"}, day).Router)
	srvNone := httptest.NewServer(NewAPI(rq, wm, []string{""}, 0).Router)
	srvNil := httptest.NewServer(NewAPI(rq, wm, nil, 0).Router)
	defer srvAll.Close()
	defer srvFoo.Close()
	defer srvNone.Close()
	defer srvNil.Close()

	var tests = [...]struct {
		name     string
		client   *http.Client
		url      string
		header   http.Header
		expected http.Header
		method   string
		max      int
	}{
		{
			name:     "simple CORS, one allowed, matching",
			client:   srvFoo.Client(),
			url:      srvFoo.URL + path,
			header:   prepareHeaders("http://foo.bar", "", "", false),
			expected: prepareHeaders("http://foo.bar", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, one allowed, scheme not matching",
			client:   srvFoo.Client(),
			url:      srvFoo.URL + path,
			header:   prepareHeaders("https://foo.bar", "", "", false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, one allowed, host not matching",
			client:   srvFoo.Client(),
			url:      srvFoo.URL + path,
			header:   prepareHeaders("http://bar.baz", "", "", false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, one allowed, port not matching",
			client:   srvFoo.Client(),
			url:      srvFoo.URL + path,
			header:   prepareHeaders("http://foo.bar:81", "", "", false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, all allowed",
			client:   srvAll.Client(),
			url:      srvAll.URL + path,
			header:   prepareHeaders("http://foo.bar", "", "", false),
			expected: prepareHeaders("*", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, none allowed",
			client:   srvNone.Client(),
			url:      srvNone.URL + path,
			header:   prepareHeaders("http://foo.bar", "", "", false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:     "simple CORS, none allowed (nil)",
			client:   srvNil.Client(),
			url:      srvNil.URL + path,
			header:   prepareHeaders("http://foo.bar", "", "", false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodGet,
		},
		{
			name:   "preflight CORS, one allowed, matching",
			client: srvFoo.Client(),
			url:    srvFoo.URL + path,
			header: prepareHeaders("http://foo.bar", contentTypeHdr, http.MethodPost,
				false),
			expected: prepareHeaders("http://foo.bar", contentTypeHdr, http.MethodPost,
				true),
			method: http.MethodOptions,
			max:    day,
		},
		{
			name:   "preflight CORS, one allowed, headers not matching",
			client: srvFoo.Client(),
			url:    srvFoo.URL + path,
			header: prepareHeaders("http://foo.bar", "Foo: Bar", http.MethodPost,
				false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodOptions,
		},
		{
			name:   "preflight CORS, one allowed, method not matching",
			client: srvFoo.Client(),
			url:    srvFoo.URL + path,
			header: prepareHeaders("http://foo.bar", contentTypeHdr, http.MethodDelete,
				false),
			expected: prepareHeaders("", "", "", true),
			method:   http.MethodOptions,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			req, err := http.NewRequest(tc.method, tc.url, nil)
			assert.Nil(err)
			for k, v := range tc.header {
				req.Header.Set(k, v[0])
			}
			resp, err := tc.client.Do(req)
			assert.Nil(err)
			for k, v := range tc.expected {
				assert.Equal(v[0], resp.Header.Get(k))
			}
			if tc.max != 0 {
				assert.Equal(strconv.Itoa(tc.max), resp.Header.Get(CORSMaxAge))
			}
		})
	}
}
