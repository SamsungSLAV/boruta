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

package client

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "git.tizen.org/tools/boruta"
	util "git.tizen.org/tools/boruta/http"
	"git.tizen.org/tools/boruta/requests"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	// name of testcase - must be the same as in server tests
	name string
	// path without server address and apiPrefix (e.g. reqs/5/prolong)
	path string
	// json that will be sent by client in HTTP request
	json string
	// expected content type
	contentType string
	// expected status
	status int
	// expected headers
	header http.Header
}

type dummyReadCloser int

func (r dummyReadCloser) Close() error {
	return errors.New("close failed")
}

func (r dummyReadCloser) Read(p []byte) (n int, err error) {
	err = errors.New("read failed")
	return
}

const (
	contentJSON = "application/json"
	dateLayout  = "2006-01-02T15:04:05Z07:00"
)

var (
	// req is valid request that may be used in tests directly or as a template.
	req            ReqInfo
	errRead        = errors.New("unable to read server response: read failed")
	errReqNotFound = util.NewServerError(NotFoundError("Request"))
)

func init() {
	deadline, err := time.Parse(dateLayout, "2200-12-31T01:02:03Z")
	if err != nil {
		panic(err)
	}
	validAfter, err := time.Parse(dateLayout, "2100-01-01T04:05:06Z")
	if err != nil {
		panic(err)
	}
	req = ReqInfo{
		Priority:   Priority(8),
		Deadline:   deadline,
		ValidAfter: validAfter,
		Caps: map[string]string{
			"architecture": "armv7l",
			"monitor":      "yes",
		},
	}
}

func initTest(t *testing.T, url string) (*assert.Assertions, *BorutaClient) {
	return assert.New(t), NewBorutaClient(url)
}

// from http/server/api/v1/api.go
func jsonMustMarshal(data interface{}) []byte {
	res, err := json.Marshal(data)
	if err != nil {
		panic("unable to marshal JSON:" + err.Error())
	}
	return res
}

// generateTestMap returns map where key is path to which client HTTP request
// will be done. Slices of testCase pointers is used as value.
func generateTestMap(tests []*testCase) map[string][]*testCase {
	ret := make(map[string][]*testCase)
	for _, test := range tests {
		ret[test.path] = append(ret[test.path], test)
	}
	return ret
}

func prepareServer(method string, tests []*testCase) *httptest.Server {
	mux := http.NewServeMux()
	tcasesMap := generateTestMap(tests)

	// Some test cases are identified only by path (e.g. GET/HEAD).
	validateTestCase := func(test []*testCase) {
		if len(test) != 1 {
			panic("len != 1")
		}
	}

	// GET or HEAD doesn't have body (and are idempotent), so path must be unique.
	if method != http.MethodPost {
		for _, v := range tcasesMap {
			validateTestCase(v)
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		var test *testCase
		var data []byte
		// Take test fixtures from server tests.
		fpath := "../server/api/v1/testdata/"
		body, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		// Making operation like "req/id/close" - path must be unique.
		if method == http.MethodPost && len(body) == 0 {
			validateTestCase(tcasesMap[r.URL.Path])
		}
		tlen := len(tcasesMap[r.URL.Path])
		if tlen == 0 {
			panic("No test cases for path: " + r.URL.Path)
		}
		for i, tcase := range tcasesMap[r.URL.Path] {
			// There may be many POST requests per one path.
			// Differentiate by body.
			if method == http.MethodPost {
				if len(body) == 0 || string(body) == tcase.json {
					test = tcase
					break
				}
				if i == tlen-1 {
					panic("matching testcase not found")
				}
			} else {
				test = tcase
				break
			}
		}
		if test.status != http.StatusNoContent {
			// Find appropriate file with reply.
			fpath += test.name + "-" + r.Method
			switch test.contentType {
			case contentJSON:
				fpath += ".json"
			default:
				fpath += ".txt"
			}
			data, err = ioutil.ReadFile(fpath)
			if err != nil {
				panic(err)
			}
			w.Header().Set("Content-Type", test.contentType)
		}
		w.WriteHeader(test.status)
		if test.status != http.StatusNoContent {
			w.Write(data)
		}
	}

	mux.Handle(apiPrefix, http.HandlerFunc(handler))
	return httptest.NewServer(mux)
}

func TestNewBorutaClient(t *testing.T) {
	assert, client := initTest(t, "")

	assert.NotNil(client)
	assert.Equal("/api/v1/", client.url)
}

func TestReadBody(t *testing.T) {
	assert := assert.New(t)
	msg := `
	W malinowym chruśniaku, przed ciekawych wzrokiem
	Zapodziani po głowy, przez długie godziny
	Zrywaliśmy przybyłe tej nocy maliny.
	Palce miałaś na oślep skrwawione ich sokiem.
	`
	reader := ioutil.NopCloser(strings.NewReader(msg))

	body, err := readBody(reader)
	assert.Nil(err)
	assert.Equal([]byte(msg), body)

	body, err = readBody(dummyReadCloser(0))
	assert.Equal(errRead, err)
	assert.Empty(body)
}

func TestBodyJSONUnmarshal(t *testing.T) {
	assert := assert.New(t)
	var reqinfo *ReqInfo
	reqJSON := jsonMustMarshal(&req)
	msg := `
	Bąk złośnik huczał basem, jakby straszył kwiaty,
	Rdzawe guzy na słońcu wygrzewał liść chory,
	Złachmaniałych pajęczyn skrzyły się wisiory,
	I szedł tyłem na grzbiecie jakiś żuk kosmaty.
	`

	reader := ioutil.NopCloser(strings.NewReader(string(reqJSON)))
	assert.Nil(bodyJSONUnmarshal(reader, &reqinfo))
	assert.Equal(&req, reqinfo)

	assert.Equal(errRead, bodyJSONUnmarshal(dummyReadCloser(0), reqinfo))

	errJSON := errors.New("unmarshalling JSON response failed: invalid character 'B' looking for beginning of value")
	reader = ioutil.NopCloser(strings.NewReader(msg))
	assert.Equal(errJSON, bodyJSONUnmarshal(reader, reqinfo))
}

func TestGetServerError(t *testing.T) {
	assert := assert.New(t)
	var resp http.Response
	resp.Header = make(http.Header)
	msg := `
	Duszno było od malin, któreś, szepcząc, rwała,
	A szept nasz tylko wówczas nacichał w ich woni,
	Gdym wargami wygarniał z podanej mi dłoni
	Owoce, przepojone wonią twego ciała.
	`

	resp.StatusCode = http.StatusOK
	assert.Nil(getServerError(&resp))

	missing := `
	{
		  "error": "Request not found"
	}
	`
	resp.Body = ioutil.NopCloser(strings.NewReader(missing))
	resp.StatusCode = http.StatusNotFound
	resp.Header.Set("Content-Type", contentJSON)
	assert.Equal(errReqNotFound, getServerError(&resp))

	resp.Body = ioutil.NopCloser(strings.NewReader(msg))
	errJSON := errors.New("unmarshalling JSON response failed: invalid character 'D' looking for beginning of value")
	assert.Equal(errJSON, getServerError(&resp))

	internal := "internal server error: test"
	resp.Body = ioutil.NopCloser(strings.NewReader(internal))
	resp.StatusCode = http.StatusInternalServerError
	resp.Header.Set("Content-Type", "text/plain")
	internalErr := util.NewServerError(util.ErrInternalServerError, "test")
	assert.Equal(internalErr, getServerError(&resp))

	resp.Body = dummyReadCloser(0)
	assert.Equal(errRead, getServerError(&resp))
}

func TestProcessResponse(t *testing.T) {
	assert := assert.New(t)
	var resp http.Response
	var reqinfo *ReqInfo
	var srvErr *util.ServerError
	missing := `
	{
		"error": "Request not found"
	}
	`

	resp.StatusCode = http.StatusNoContent
	assert.Nil(processResponse(&resp, &reqinfo))
	assert.Nil(reqinfo)

	reqinfo = new(ReqInfo)
	assert.Nil(processResponse(&resp, &reqinfo))
	assert.Nil(reqinfo)

	resp.Header = make(http.Header)
	resp.Header.Set("Content-Type", contentJSON)

	resp.StatusCode = http.StatusOK
	resp.Body = ioutil.NopCloser(strings.NewReader(string(jsonMustMarshal(&req))))
	assert.Nil(processResponse(&resp, &reqinfo))
	assert.Equal(&req, reqinfo)

	resp.StatusCode = http.StatusNotFound
	resp.Body = ioutil.NopCloser(strings.NewReader(missing))
	srvErr = processResponse(&resp, &reqinfo).(*util.ServerError)
	assert.Equal(errReqNotFound, srvErr)

	badType := "can't set val, please pass appropriate pointer"
	var foo int
	assert.PanicsWithValue(badType, func() { processResponse(&resp, foo) })
}

func TestNewRequest(t *testing.T) {
	prefix := "new-req-"
	path := "/api/v1/reqs/"

	badPrio := req
	badPrio.Priority = Priority(32)
	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix + "valid",
			path:        path,
			json:        string(jsonMustMarshal(&req)),
			contentType: contentJSON,
			status:      http.StatusCreated,
		},
		&testCase{
			// bad request - priority out of bounds
			name:        prefix + "bad-prio",
			path:        path,
			json:        string(jsonMustMarshal(&badPrio)),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid request
	reqID, err := client.NewRequest(req.Caps, req.Priority,
		req.Owner, req.ValidAfter, req.Deadline)
	assert.Equal(ReqID(1), reqID)
	assert.Nil(err)

	// bad request - priority out of bounds
	expectedErr := util.NewServerError(requests.ErrPriority)
	reqID, err = client.NewRequest(badPrio.Caps, badPrio.Priority,
		badPrio.Owner, badPrio.ValidAfter, badPrio.Deadline)
	assert.Zero(reqID)
	assert.Equal(expectedErr, err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	reqID, err = client.NewRequest(req.Caps, req.Priority,
		req.Owner, req.ValidAfter, req.Deadline)
	assert.Zero(reqID)
	assert.NotNil(err)
}

func TestCloseRequest(t *testing.T) {
	prefix := "close-req-"
	path := "/api/v1/reqs/"
	tests := []*testCase{
		&testCase{
			// valid request
			name:   prefix + "valid",
			path:   path + "1" + "/close",
			status: http.StatusNoContent,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "2" + "/close",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid request
	assert.Nil(client.CloseRequest(ReqID(1)))

	// missing request
	assert.Equal(errReqNotFound, client.CloseRequest(ReqID(2)))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.CloseRequest(ReqID(1)))
}

func TestUpdateRequest(t *testing.T) {
	prefix := "update-req-"
	path := "/api/v1/reqs/"

	validAfter := time.Now()
	deadline := validAfter.AddDate(0, 0, 2)
	priority := Priority(4)

	reqJSON := string(jsonMustMarshal(&struct {
		Priority
		Deadline   time.Time
		ValidAfter time.Time
	}{
		Priority:   priority,
		Deadline:   deadline,
		ValidAfter: validAfter,
	}))

	reqinfo := req

	reqinfo.Priority = priority
	reqinfo.Deadline = deadline
	reqinfo.ValidAfter = validAfter

	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix + "valid",
			path:        path + "1",
			json:        reqJSON,
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "2",
			json:        reqJSON,
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	reqinfo.ID = ReqID(1)
	assert.Nil(client.UpdateRequest(&reqinfo))

	// missing
	reqinfo.ID = ReqID(2)
	assert.Equal(errReqNotFound, client.UpdateRequest(&reqinfo))

	// bad arguments
	err := client.UpdateRequest(nil)
	assert.Equal(errors.New("nil reqInfo passed"), err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.UpdateRequest(&reqinfo))
}

func TestGetRequestInfo(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	reqInfo, err := client.GetRequestInfo(ReqID(0))
	assert.Nil(reqInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestListRequests(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	reqInfo, err := client.ListRequests(nil)
	assert.Nil(reqInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestAcquireWorker(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	accessInfo, err := client.AcquireWorker(ReqID(0))
	assert.Nil(accessInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestProlongAccess(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	err := client.ProlongAccess(ReqID(0))
	assert.Equal(util.ErrNotImplemented, err)
}

func TestListWorkers(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	list, err := client.ListWorkers(nil, nil)
	assert.Nil(list)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestGetWorkerInfo(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	info, err := client.GetWorkerInfo(WorkerUUID(""))
	assert.Zero(info)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestSetState(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	err := client.SetState(WorkerUUID(""), FAIL)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestSetGroups(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	err := client.SetGroups(WorkerUUID(""), nil)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestDeregister(t *testing.T) {
	assert, client := initTest(t, "")
	assert.NotNil(client)

	err := client.Deregister(WorkerUUID(""))
	assert.Equal(util.ErrNotImplemented, err)
}
