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
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/mocks"
	"github.com/dimfeld/httptreemux"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	contentTypeJSON = "application/json"
	invalidID       = "test"
	dateLayout      = "2006-01-02"
	past            = "1683-09-12"
	future          = "2222-12-31"
	validUUID       = "ec4898ac-0853-407c-8501-cbb24ef6bd77"
	missingUUID     = "8f8ade90-a319-4275-9407-977ca3e9607c"
	validReqJSON    = `{
		"ID":1,
		"State":"WAITING",
		"Job":null,
		"Priority":8,
		"Deadline":"2200-12-31T01:02:03Z",
		"ValidAfter":"2100-01-01T04:05:06Z",
		"Caps":{
			"architecture":"armv7l",
			"monitor":"yes"
		},
		"Owner":{}
	}`
)

var update bool

type requestTest struct {
	name        string
	path        string
	methods     []string
	json        string
	contentType string
	status      int
	header      http.Header
}

type allMocks struct {
	ctrl *gomock.Controller
	rq   *mocks.MockRequests
	wm   *mocks.MockWorkers
}

// TestTempl variables shouldn't be used directly, but rather as an input for
// testFromTempl() function.
var (
	// malformedJSONTestTempl may be used by functions that need to check
	// for malformed JSON to initialize test. Every test must set path, name
	// and method appropriately.
	malformedJSONTestTempl = &requestTest{
		name:        "malformed-json",
		path:        "",
		methods:     []string{},
		json:        `{"Priority{}`,
		contentType: contentTypeJSON,
		status:      http.StatusBadRequest,
	}

	// invalidIDTestTempl may be used by functions that need to check for
	// cases where malformed ID is given in a URL to initialize test.
	// Every test must set path, name and method appropriately.
	invalidIDTestTempl = &requestTest{
		name:        "bad-id",
		path:        "",
		methods:     []string{},
		json:        ``,
		contentType: contentTypeJSON,
		status:      http.StatusBadRequest,
	}

	// notFoundTestTempl may be used by functions that need to check for
	// not existing requests to initialize test. Every test must set path,
	// name and method appropriately.
	notFoundTestTempl = &requestTest{
		name:        "missing",
		path:        "",
		methods:     []string{},
		json:        ``,
		contentType: contentTypeJSON,
		status:      http.StatusNotFound,
	}
)

func TestMain(m *testing.M) {
	flag.BoolVar(&update, "update", false, "update testdata")
	flag.Parse()
	os.Exit(m.Run())
}

func initTest(t *testing.T) (*assert.Assertions, *allMocks, *httptreemux.TreeMux) {
	r := httptreemux.New()
	ctrl := gomock.NewController(t)
	m := &allMocks{
		ctrl: ctrl,
		rq:   mocks.NewMockRequests(ctrl),
		wm:   mocks.NewMockWorkers(ctrl),
	}
	_ = NewAPI(r.NewGroup("/api/"+Version), m.rq, m.wm)
	return assert.New(t), m, r
}

func (m *allMocks) finish() {
	m.ctrl.Finish()
}

func testFromTempl(templ *requestTest, name string, path string,
	methods ...string) (ret requestTest) {
	ret = *templ
	ret.name = name + templ.name
	ret.path = path
	if len(methods) != 0 {
		ret.methods = methods
	}
	return
}

func newWorker(uuid string, state WorkerState, groups Groups, caps Capabilities) (w WorkerInfo) {
	if caps == nil {
		caps = make(Capabilities)
	}
	caps["UUID"] = uuid
	w = WorkerInfo{
		WorkerUUID: WorkerUUID(uuid),
		State:      state,
		Caps:       caps,
	}
	if len(groups) != 0 {
		w.Groups = groups
	}
	return
}

func runTests(assert *assert.Assertions, r *httptreemux.TreeMux, tests []requestTest) {
	srv := httptest.NewServer(r)
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
			if update && resp.StatusCode != http.StatusNoContent &&
				method != http.MethodHead {
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
			// check Boruta HTTP headers
			if test.header != nil && len(test.header) > 0 {
				for k, v := range test.header {
					assert.Contains(resp.Header, k, tcaseErrStr+" (header)")
					assert.Equal(v, resp.Header[k], tcaseErrStr+" (header)")
				}
			}
			// check result JSON
			expected, err := ioutil.ReadFile(tdata)
			assert.Nil(err, tcaseErrStr)
			assert.JSONEq(string(expected), string(body), tcaseErrStr+" (JSON)")
		}
	}
}

func TestNewAPI(t *testing.T) {
	assert, m, api := initTest(t)
	assert.NotNil(api)
	m.finish()
}

func TestJsonMustMarshal(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() { jsonMustMarshal(make(chan bool)) })
}

func TestParseReqID(t *testing.T) {
	assert := assert.New(t)
	reqid, err := parseReqID("1")
	assert.Nil(err)
	assert.Equal(ReqID(1), reqid)
	_, err = parseReqID(invalidID)
	assert.NotNil(err)
}

func TestIsValidUUID(t *testing.T) {
	assert := assert.New(t)
	assert.True(isValidUUID(validUUID))
	assert.False(isValidUUID(invalidID))
}
