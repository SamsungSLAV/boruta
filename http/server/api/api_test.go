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
	"testing"

	. "git.tizen.org/tools/boruta"
	util "git.tizen.org/tools/boruta/http"
	"git.tizen.org/tools/boruta/matcher"
	"git.tizen.org/tools/boruta/requests"
	"git.tizen.org/tools/boruta/workers"
	"github.com/dimfeld/httptreemux"
	"github.com/stretchr/testify/assert"
)

var update bool

func TestMain(m *testing.M) {
	flag.BoolVar(&update, "update", false, "update testdata")
	flag.Parse()
	os.Exit(m.Run())
}

func initTest(t *testing.T) (*assert.Assertions, *API) {
	wm := workers.NewWorkerList()
	return assert.New(t), NewAPI(httptreemux.New(),
		requests.NewRequestQueue(wm, matcher.NewJobsManager(wm)), wm)
}

func TestPanicHandler(t *testing.T) {
	assert, api := initTest(t)
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
	srv := httptest.NewServer(api.r)
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
	assert, api := initTest(t)
	assert.NotNil(api)
}

func TestRedirectToDefault(t *testing.T) {
	assert, api := initTest(t)
	srv := httptest.NewServer(api.r)
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
	assert.Equal(NotFoundError(badPath).Error(), srvErr.Err)
}
