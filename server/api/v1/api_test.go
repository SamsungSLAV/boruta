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

package v1

import (
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.tizen.org/tools/boruta/mocks"
	"github.com/dimfeld/httptreemux"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const contentTypeJSON = "application/json"

var update bool

type requestTest struct {
	name        string
	path        string
	methods     []string
	json        string
	contentType string
	status      int
}

type allMocks struct {
	ctrl *gomock.Controller
	rq   *mocks.MockRequests
	wm   *mocks.MockWorkers
}

func TestMain(m *testing.M) {
	flag.BoolVar(&update, "update", false, "update testdata")
	flag.Parse()
	os.Exit(m.Run())
}

func initTest(t *testing.T) (*assert.Assertions, *allMocks, *API) {
	ctrl := gomock.NewController(t)
	m := &allMocks{
		ctrl: ctrl,
		rq:   mocks.NewMockRequests(ctrl),
		wm:   mocks.NewMockWorkers(ctrl),
	}
	return assert.New(t), m, NewAPI(httptreemux.New())
}

func (m *allMocks) finish() {
	m.ctrl.Finish()
}

func runTests(assert *assert.Assertions, api *API, tests []requestTest) {
	srv := httptest.NewServer(api.r)
	defer srv.Close()
	var req *http.Request
	var err error
	var tcaseErrStr string

	for _, test := range tests {
		tcaseErrStr = test.name + ": FAILED"
		for _, method := range test.methods {
			// prepare and do HTTP request
			if test.json == "" {
				req, err = http.NewRequest(method,
					srv.URL+test.path, nil)
			} else {
				req, err = http.NewRequest(method,
					srv.URL+test.path,
					strings.NewReader(test.json))
			}
			assert.Nil(err)
			req.Header["Content-Type"] = []string{test.contentType}
			resp, err := srv.Client().Do(req)
			assert.Nil(err)
			defer resp.Body.Close()

			// read expected results from file or generate the file
			tdata := filepath.Join("testdata", test.name+"-"+method+".json")
			body, err := ioutil.ReadAll(resp.Body)
			assert.Nil(err)
			if update && method != http.MethodHead {
				err = ioutil.WriteFile(tdata, body, 0644)
				assert.Nil(err)
			}

			// check status code
			assert.Equal(test.status, resp.StatusCode, tcaseErrStr)
			if resp.StatusCode == http.StatusNoContent {
				continue
			}
			// check content type
			assert.Equal(test.contentType,
				resp.Header.Get("Content-Type"))
			if method == http.MethodHead {
				assert.Zero(len(body), tcaseErrStr)
				continue
			}
			// if update was set then file was just generated,
			// so there's no sense in rereading and comparing it.
			if update {
				continue
			}
			// check result JSON
			expected, err := ioutil.ReadFile(tdata)
			assert.Nil(err, tcaseErrStr)
			assert.JSONEq(string(expected), string(body), tcaseErrStr)
		}
	}
}

func TestNewAPI(t *testing.T) {
	assert, m, api := initTest(t)
	assert.NotNil(api)
	m.finish()
}

func TestNewServerError(t *testing.T) {
	assert := assert.New(t)
	badRequest := &serverError{
		Err:    "invalid request: foo",
		Status: http.StatusBadRequest,
	}
	nobody := "no body provided in HTTP request"
	missingBody := &serverError{
		Err:    nobody,
		Status: http.StatusBadRequest,
	}
	notImplemented := &serverError{
		Err:    ErrNotImplemented.Error(),
		Status: http.StatusNotImplemented,
	}
	internalErr := &serverError{
		Err:    ErrInternalServerError.Error(),
		Status: http.StatusInternalServerError,
	}
	customErr := &serverError{
		Err:    "invalid request: more details",
		Status: http.StatusBadRequest,
	}
	assert.Equal(badRequest, newServerError(errors.New("foo")))
	assert.Equal(missingBody, newServerError(io.EOF))
	assert.Equal(notImplemented, newServerError(ErrNotImplemented))
	assert.Equal(internalErr, newServerError(ErrInternalServerError))
	assert.Equal(customErr, newServerError(ErrBadRequest, "more details"))
	assert.Nil(newServerError(nil))
}

func TestJsonMustMarshal(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() { jsonMustMarshal(make(chan bool)) })
}

func TestPanicHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()
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
			err: &serverError{
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

	newHandler := func(err interface{}) reqHandler {
		return func(r *http.Request, ps map[string]string) responseData {
			panic(err)
		}
	}
	for _, test := range tests {
		routerSetHandler(api.r.NewGroup("/"), test.path, newHandler(test.err),
			http.StatusOK, http.MethodGet)
	}
	var tcaseErrStr string
	srv := httptest.NewServer(api.r)
	assert.NotNil(srv)
	defer srv.Close()
	for _, test := range tests {
		tcaseErrStr = test.name + ": FAILED"
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
		assert.Equal(http.StatusInternalServerError, resp.StatusCode, tcaseErrStr)
		assert.Equal(contentType, resp.Header.Get("Content-Type"), tcaseErrStr)
		assert.Equal(expected, body, tcaseErrStr)
	}
}
