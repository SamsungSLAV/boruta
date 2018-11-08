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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	util "github.com/SamsungSLAV/boruta/http"
	"github.com/SamsungSLAV/boruta/requests"
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
	// response that should be returned (read from server testdata if empty)
	resp string
	// expected headers
	header http.Header
}

type dummyListFilter bool

func (*dummyListFilter) Match(_ interface{}) bool { return false }

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
	invalidID   = "test"
	validUUID   = "ec4898ac-0853-407c-8501-cbb24ef6bd77"
)

var (
	// req is valid request that may be used in tests directly or as a template.
	req            boruta.ReqInfo
	errRead        = errors.New("unable to read server response: read failed")
	errReqNotFound = util.NewServerError(boruta.NotFoundError("Request"))
	sorterBad      = &boruta.SortInfo{
		Item:  "foobarbaz",
		Order: boruta.SortOrderDesc,
	}
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
	req = boruta.ReqInfo{
		Priority:   boruta.Priority(8),
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
			validateTestCase(tcasesMap[r.URL.String()])
		}
		tlen := len(tcasesMap[r.URL.String()])
		if tlen == 0 {
			panic("No test cases for path: " + r.URL.String())
		}
		for i, tcase := range tcasesMap[r.URL.String()] {
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
		if test.status != http.StatusNoContent && r.Method != http.MethodHead {
			// Find appropriate file with reply.
			fpath += test.name + "-" + r.Method
			switch test.contentType {
			case contentJSON:
				fpath += ".json"
			default:
				fpath += ".txt"
			}
			// Or use reply provided in testcase instead.
			if test.resp == "" {
				data, err = ioutil.ReadFile(fpath)
				if err != nil {
					panic(err)
				}
			} else {
				data = []byte(test.resp)
			}
			w.Header().Set("Content-Type", test.contentType)
		}
		// Set custom Boruta HTTP headers.
		if test.header != nil {
			for k := range test.header {
				w.Header().Set(k, test.header.Get(k))
			}
		}
		w.WriteHeader(test.status)
		if test.status != http.StatusNoContent {
			w.Write(data)
		}
	}

	mux.Handle(apiPrefix, http.HandlerFunc(handler))
	return httptest.NewServer(mux)
}

// from http/server/api/v1/api_test.go
func newWorker(uuid string, state boruta.WorkerState, groups boruta.Groups,
	caps boruta.Capabilities) (w boruta.WorkerInfo) {

	if caps == nil {
		caps = make(boruta.Capabilities)
	}
	caps["UUID"] = uuid
	w = boruta.WorkerInfo{
		WorkerUUID: boruta.WorkerUUID(uuid),
		State:      state,
		Caps:       caps,
	}
	if len(groups) != 0 {
		w.Groups = groups
	}
	return
}

func updateHeaders(hdr http.Header, info *boruta.ListInfo) (ret http.Header) {
	ret = make(http.Header)
	for k := range hdr {
		copy(ret[k], hdr[k])
	}
	if info != nil {
		ret.Set(util.ListTotalItemsHdr, strconv.FormatUint(info.TotalItems, 10))
		ret.Set(util.ListRemainingItemsHdr, strconv.FormatUint(info.RemainingItems, 10))
	}
	return
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
	var reqinfo *boruta.ReqInfo
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
		  "message": "Request not found"
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
	var reqinfo *boruta.ReqInfo
	var srvErr *util.ServerError
	missing := `
	{
		"message": "Request not found"
	}
	`

	resp.StatusCode = http.StatusNoContent
	assert.Nil(processResponse(&resp, &reqinfo))
	assert.Nil(reqinfo)

	reqinfo = new(boruta.ReqInfo)
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

func TestCheckStatus(t *testing.T) {
	var resp http.Response
	resp.StatusCode = http.StatusNoContent
	err := errors.New("bad HTTP status: 204 No Content")

	assert := assert.New(t)

	assert.Nil(checkStatus(http.StatusNoContent, &resp))
	resp.Status = "204 No Content"
	assert.Equal(err, checkStatus(http.StatusBadRequest, &resp))
}

func TestGetHeaders(t *testing.T) {
	prefix := "boruta-headers-"
	pathW := "/api/v1/workers/"
	pathR := "/api/v1/reqs/"
	date := time.Now().Format(util.DateFormat)

	worker := make(http.Header)
	worker.Set("Boruta-Worker-State", string(boruta.RUN))

	request := make(http.Header)
	request.Set("Boruta-Request-State", string(boruta.INPROGRESS))
	request.Set("Boruta-Job-Timeout", date)

	tests := []*testCase{
		&testCase{
			// valid worker
			name:   prefix + "worker",
			path:   pathW + validUUID,
			status: http.StatusNoContent,
			header: worker,
		},
		&testCase{
			// valid request
			name:   prefix + "request",
			path:   pathR + "1",
			status: http.StatusNoContent,
			header: request,
		},

		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        pathW + invalidID,
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	assert := assert.New(t)
	srv := prepareServer(http.MethodHead, tests)
	url := srv.URL
	defer srv.Close()

	// valid worker
	headers, err := getHeaders(url + pathW + validUUID)
	assert.Nil(err)
	assert.Equal(worker.Get("Boruta-Worker-State"), headers.Get("Boruta-Worker-State"))

	// valid request
	headers, err = getHeaders(url + pathR + "1")
	assert.Nil(err)
	assert.Equal(request.Get("Boruta-Request-State"), headers.Get("Boruta-Request-State"))
	assert.Equal(request.Get("Boruta-Job-Timeout"), headers.Get("Boruta-Job-Timeout"))

	// invalid UUID
	headers, err = getHeaders(url + pathW + invalidID)
	assert.Nil(headers)
	assert.Equal(errors.New("bad HTTP status: 400 Bad Request"), err)

	// http.Head failure
	url = "http://nosuchaddress.fail"
	headers, err = getHeaders(url)
	assert.Nil(headers)
	assert.NotNil(err)

}

func TestNewRequest(t *testing.T) {
	prefix := "new-req-"
	path := "/api/v1/reqs/"

	badPrio := req
	badPrio.Priority = boruta.Priority(32)
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
	assert.Equal(boruta.ReqID(1), reqID)
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
	assert.Nil(client.CloseRequest(boruta.ReqID(1)))

	// missing request
	assert.Equal(errReqNotFound, client.CloseRequest(boruta.ReqID(2)))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.CloseRequest(boruta.ReqID(1)))
}

func TestUpdateRequest(t *testing.T) {
	prefix := "update-req-"
	path := "/api/v1/reqs/"

	validAfter := time.Now()
	deadline := validAfter.AddDate(0, 0, 2)
	priority := boruta.Priority(4)

	reqJSON := string(jsonMustMarshal(&struct {
		boruta.Priority
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
	reqinfo.ID = boruta.ReqID(1)
	assert.Nil(client.UpdateRequest(&reqinfo))

	// missing
	reqinfo.ID = boruta.ReqID(2)
	assert.Equal(errReqNotFound, client.UpdateRequest(&reqinfo))

	// bad arguments
	err := client.UpdateRequest(nil)
	assert.Equal(errors.New("nil reqInfo passed"), err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.UpdateRequest(&reqinfo))
}

func TestGetRequestInfo(t *testing.T) {
	prefix := "req-info"
	path := "/api/v1/reqs/"

	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix,
			path:        path + "1",
			contentType: contentJSON,
			status:      http.StatusOK,
		},
		&testCase{
			// missing request
			name:        prefix + "-missing",
			path:        path + "2",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
	}

	srv := prepareServer(http.MethodGet, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	req := req
	req.ID = boruta.ReqID(1)
	req.State = boruta.WAIT
	reqInfo, err := client.GetRequestInfo(boruta.ReqID(1))
	assert.Nil(err)
	assert.Equal(req, reqInfo)

	// missing
	reqInfo, err = client.GetRequestInfo(boruta.ReqID(2))
	assert.Zero(reqInfo)
	assert.Equal(errReqNotFound, err)

	// http.Get failure
	client.url = "http://nosuchaddress.fail"
	reqInfo, err = client.GetRequestInfo(boruta.ReqID(1))
	assert.Zero(reqInfo)
	assert.NotNil(err)
}

func TestListRequests(t *testing.T) {
	prefix := "filter-reqs-"
	path := "/api/v1/reqs/list"

	// from api/v1 TestListRequestsHandler
	deadline, _ := time.Parse(dateLayout, "2222-12-31T00:00:00Z")
	validAfter, _ := time.Parse(dateLayout, "1683-09-12T00:00:00Z")
	reqs := []boruta.ReqInfo{
		{ID: 1, Priority: (boruta.HiPrio + boruta.LoPrio) / 2, State: boruta.WAIT,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 2, Priority: (boruta.HiPrio+boruta.LoPrio)/2 + 1, State: boruta.WAIT,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 3, Priority: (boruta.HiPrio + boruta.LoPrio) / 2, State: boruta.CANCEL,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 4, Priority: (boruta.HiPrio+boruta.LoPrio)/2 + 1, State: boruta.CANCEL,
			Deadline: deadline, ValidAfter: validAfter},
	}

	validInfo := &boruta.ListInfo{TotalItems: 2, RemainingItems: 0}
	firstInfo := &boruta.ListInfo{TotalItems: 4, RemainingItems: 2}
	secondInfo := &boruta.ListInfo{TotalItems: 4, RemainingItems: 0}

	fwdPaginator1 := &boruta.RequestsPaginator{
		ID:        0,
		Direction: boruta.DirectionForward,
		Limit:     2,
	}
	fwdPaginator2 := &boruta.RequestsPaginator{
		ID:        2,
		Direction: boruta.DirectionForward,
		Limit:     2,
	}
	backPaginator1 := &boruta.RequestsPaginator{
		ID:        4,
		Direction: boruta.DirectionBackward,
		Limit:     2,
	}
	backPaginator2 := &boruta.RequestsPaginator{
		ID:        2,
		Direction: boruta.DirectionBackward,
		Limit:     2,
	}

	missingFilter := filter.NewRequests("INPROGRESS", "2")
	missingHeader := make(http.Header)
	missingHeader.Set(util.RequestCountHdr, "0")
	validFilter := filter.NewRequests("WAIT", "")
	validHeader := make(http.Header)
	validHeader.Set(util.RequestCountHdr, "2")
	nilHeader := make(http.Header)
	nilHeader.Set(util.RequestCountHdr, "4")
	malformedHeader := make(http.Header)
	malformedHeader.Set(util.ListTotalItemsHdr, "foo")

	sorterDefault := new(boruta.SortInfo) // "", SortOrderAsc
	sorterAsc := &boruta.SortInfo{
		Item:  "priority",
		Order: boruta.SortOrderAsc,
	}
	sorterDesc := &boruta.SortInfo{
		Item:  "state",
		Order: boruta.SortOrderDesc,
	}

	tests := []*testCase{
		&testCase{
			// valid filter
			name: prefix + "valid-filter",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterDefault,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(validHeader, validInfo),
		},
		&testCase{
			// Valid filter with sorter - list some requests sorted by priority (ascending).
			name: prefix + "valid-filter-sort-asc",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterAsc,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(validHeader, validInfo),
		},
		&testCase{
			// Valid filter with sorter - list some requests sorted by state (descending).
			// As state is equal - this will be sorted by ID.
			name: prefix + "valid-filter-sort-desc",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterDesc,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(validHeader, validInfo),
		},
		&testCase{
			// nil filter - list all
			name:        prefix + "nil",
			path:        path,
			json:        string(jsonMustMarshal(&util.RequestsListSpec{})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(nilHeader, secondInfo),
		},
		// List first part of all requests.
		{
			name: prefix + "paginator-fwd1",
			path: path,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Paginator: fwdPaginator1,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(nilHeader, firstInfo),
		},
		// List second part of all requests.
		{
			name: prefix + "paginator-fwd2",
			path: path,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Paginator: fwdPaginator2,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(nilHeader, secondInfo),
		},
		// List first part of all requests (backward).
		{
			name: prefix + "paginator-back1",
			path: path,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Paginator: backPaginator1,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(nilHeader, firstInfo),
		},
		// List second part of all requests (backward).
		{
			name: prefix + "paginator-back2",
			path: path,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Paginator: backPaginator2,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(nilHeader, secondInfo),
		},
		&testCase{
			// no requests matched
			name: prefix + "nomatch-all",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: missingFilter,
				Sorter: sorterDefault,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      updateHeaders(missingHeader, &boruta.ListInfo{}),
		},
		&testCase{
			// missing headers
			name: prefix + "missing-headers",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: missingFilter,
				Sorter: sorterAsc,
			})),
			resp:        string(jsonMustMarshal(nil)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
		&testCase{
			// malformed headers
			name: prefix + "malformed-headers",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Sorter: sorterAsc,
			})),
			resp:        string(jsonMustMarshal(nil)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      malformedHeader,
		},
		&testCase{
			// Bad sort item.
			name: prefix + "bad-sorter-item",
			path: path,
			json: string(jsonMustMarshal(&util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterBad,
			})),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	sorter := new(boruta.SortInfo)

	// nil spec
	list, info, err := client.ListRequests(nil, nil, nil)
	assert.Nil(err)
	assert.Equal(reqs, list)
	assert.Equal(secondInfo, info)

	// valid filter
	list, info, err = client.ListRequests(validFilter, sorter, nil)
	assert.Nil(err)
	assert.Equal(reqs[:2], list)
	assert.Equal(validInfo, info)

	// no requests matched
	list, info, err = client.ListRequests(missingFilter, sorter, nil)
	assert.Nil(err)
	assert.Equal([]boruta.ReqInfo{}, list)
	assert.Equal(&boruta.ListInfo{}, info)

	// valid filter, sort by priority, ascending.
	sorter.Item = "priority"
	list, info, err = client.ListRequests(validFilter, sorter, nil)
	assert.Nil(err)
	assert.Equal(reqs[:2], list)
	assert.Equal(validInfo, info)

	// valid filter, sort by state, descending.
	sorter.Item = "state"
	sorter.Order = boruta.SortOrderDesc
	list, info, err = client.ListRequests(validFilter, sorter, nil)
	assert.Nil(err)
	assert.Equal([]boruta.ReqInfo{reqs[1], reqs[0]}, list)
	assert.Equal(validInfo, info)

	// List first part of all requests.
	list, info, err = client.ListRequests(nil, nil, fwdPaginator1)
	assert.Nil(err)
	assert.Equal(reqs[:2], list)
	assert.Equal(firstInfo, info)

	// List second part of all requests.
	list, info, err = client.ListRequests(nil, nil, fwdPaginator2)
	assert.Nil(err)
	assert.Equal(reqs[2:], list)
	assert.Equal(secondInfo, info)

	// List first part of all requests (backward).
	list, info, err = client.ListRequests(nil, nil, backPaginator1)
	assert.Nil(err)
	assert.Equal(reqs[2:], list)
	assert.Equal(firstInfo, info)

	// List second part of all requests (backward).
	list, info, err = client.ListRequests(nil, nil, backPaginator2)
	assert.Nil(err)
	assert.Equal(reqs[:2], list)
	assert.Equal(secondInfo, info)

	// Missing headers.
	list, info, err = client.ListRequests(missingFilter, sorterAsc, nil)
	assert.Nil(list)
	assert.Nil(info)
	assert.NotNil(err)

	// Malformed headers.
	list, info, err = client.ListRequests(nil, sorterAsc, nil)
	assert.Nil(list)
	assert.Nil(info)
	assert.NotNil(err)

	// Bad sort item.
	sorter.Item = "foobarbaz"
	list, info, err = client.ListRequests(validFilter, sorter, nil)
	assert.Empty(list)
	assert.Nil(info)
	assert.NotNil(err)

	// Wrong type of filter.
	list, info, err = client.ListRequests(new(dummyListFilter), sorter, nil)
	assert.Empty(list)
	assert.Nil(info)
	assert.NotNil(err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	list, info, err = client.ListRequests(nil, sorter, nil)
	assert.Zero(list)
	assert.Nil(info)
	assert.NotNil(err)
}

func TestAcquireWorker(t *testing.T) {
	prefix := "acquire-worker-"
	path := "/api/v1/reqs/"

	badkeyAI := util.AccessInfo2{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 22,
		},
		Key:      "bad key :(",
		Username: "lelpolel",
	}

	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix + "valid",
			path:        path + "1/acquire_worker",
			contentType: contentJSON,
			status:      http.StatusOK,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "2/acquire_worker",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
		&testCase{
			// bad key request
			name:        prefix + "badkey",
			path:        path + "3/acquire_worker",
			contentType: contentJSON,
			status:      http.StatusOK,
			resp:        string(jsonMustMarshal(badkeyAI)),
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// from server/api/v1 AcquireWorker tests
	keyPem := "-----BEGIN RSA PRIVATE KEY-----\nMIICXgIBAAKBgQCyBgKbrwKh75BDoigwltbazFGDLdlxf9YLpFj5v+4ieKgsaN+W\n+kRvamSuB5CC2tqFql5x7kPt1U+vVMwkzVRewF/HHzRYxgLHlge6d1ZALpCWywaz\nslt5pNCmF7NoZ//WTSrafufDI4IRoNgkHtEKvnWdBaPPnY4Cf+PCbZOYNQIDAQAB\nAoGBAJvoz5fxKekQmdPhzDjhocF1d13fZbQVNSx0/seb476k1QQvxMHA5PZ+wzX2\nwgUYDpFJp/U3qp48VtFC/pasjNoG7zLPLLUJcg15eOoh4Ld7I1e4lRkLl3CwnqMk\nbc6UoKQRLli4O3cmaMxVHXal0o72s3o0qnHlRlZXLekwi6aBAkEA69j3bnbAybsF\n/NHelRYDH8bou+LCX2d/p6ReUR0bJ4yCAWRi/9Ld0ng482xinxGSpfovbIplBMFx\nH2eT2Cw0OQJBAME8LLz/3zb/vLG/t8Lfsequ1ZhVca/LlVR4yJLlyaVcywT9SJlO\nmKCy13SpKl8TY7czyufYrY4lobZjYaIsm90CQQCKhkRGWG/BzRymMyp2DJjHKFB4\nUqbx3FuJPqy7HcpeP1P4t1rCgbsSLNTefRGr9mlZHYqPSPYuheQImxCmTshZAkEA\nwAp5u+vfft1yPoT2r+l4/G99P8PLFJcTdbwEOlm8qWcrLW47dIE0FqEml3536b1v\nYGdMxFYHRjoIGSdzpKUI0QJAAqPdDp+y7kWaeIbKkp3Z3bLrj5wii2QAy2YlBDKe\npXrvruWJvL75OCYcxRju3DpVaoYqEmso+UEiQEDRB42YYg==\n-----END RSA PRIVATE KEY-----\n"
	block, _ := pem.Decode([]byte(keyPem))
	assert.NotNil(block)
	key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	access := boruta.AccessInfo{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 22,
		},
		Key:      *key,
		Username: "wołchw",
	}

	// valid
	accessInfo, err := client.AcquireWorker(boruta.ReqID(1))
	assert.Equal(access, accessInfo)
	assert.Nil(err)

	// missing
	accessInfo, err = client.AcquireWorker(boruta.ReqID(2))
	assert.Zero(accessInfo)
	assert.Equal(errReqNotFound, err)

	// bad key
	accessInfo, err = client.AcquireWorker(boruta.ReqID(3))
	assert.Zero(accessInfo)
	assert.Equal(errors.New("wrong key: "+badkeyAI.Key), err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	accessInfo, err = client.AcquireWorker(boruta.ReqID(1))
	assert.Zero(accessInfo)
	assert.NotNil(err)
}

func TestProlongAccess(t *testing.T) {
	prefix := "prolong-access-"
	path := "/api/v1/reqs/"

	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix + "valid",
			path:        path + "1/prolong",
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "2/prolong",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	assert.Nil(client.ProlongAccess(boruta.ReqID(1)))

	// missing
	assert.Equal(errReqNotFound, client.ProlongAccess(boruta.ReqID(2)))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.ProlongAccess(boruta.ReqID(1)))
}

func TestListWorkers(t *testing.T) {
	prefix := "filter-workers-"
	path := "/api/v1/workers/list"

	// based on http/server/api/v1/handlers_test.go
	armCaps := make(boruta.Capabilities)
	armCaps["architecture"] = "AArch64"
	riscvCaps := make(boruta.Capabilities)
	riscvCaps["architecture"] = "RISC-V"
	workers := []boruta.WorkerInfo{
		newWorker("0", boruta.IDLE, boruta.Groups{"Lędzianie"}, armCaps),
		newWorker("1", boruta.FAIL, boruta.Groups{"Malinowy Chruśniak"}, armCaps),
		newWorker("2", boruta.IDLE, boruta.Groups{"Malinowy Chruśniak", "Lędzianie"}, riscvCaps),
		newWorker("3", boruta.FAIL, boruta.Groups{"Malinowy Chruśniak"}, riscvCaps),
	}
	sorterAsc := &boruta.SortInfo{
		Item:  "UUID",
		Order: boruta.SortOrderAsc,
	}
	sorterDesc := &boruta.SortInfo{
		Item:  "state",
		Order: boruta.SortOrderDesc,
	}
	validFilter := &filter.Workers{
		Groups:       boruta.Groups{"Lędzianie"},
		Capabilities: map[string]string{"architecture": "AArch64"},
	}
	validHeader := make(http.Header)
	validHeader.Set(util.WorkerCountHdr, "2")
	allHeader := make(http.Header)
	allHeader.Set(util.WorkerCountHdr, "4")
	missingFilter := &filter.Workers{
		Groups: boruta.Groups{"Fern Flower"},
	}
	missingHeader := make(http.Header)
	missingHeader.Set(util.WorkerCountHdr, "0")

	tests := []*testCase{
		&testCase{
			// valid request
			name: prefix + "valid-filter",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: validFilter,
				Sorter: &boruta.SortInfo{},
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      validHeader,
		},
		&testCase{
			// list all
			name: prefix + "empty-filter",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: nil,
				Sorter: nil,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		&testCase{
			// list all - sorted by uuid, ascending
			name: prefix + "empty-filter-sort-asc",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: nil,
				Sorter: sorterAsc,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		&testCase{
			// list all - sorted by state, descending
			name: prefix + "empty-filter-sort-desc",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: nil,
				Sorter: sorterDesc,
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		&testCase{
			// no matches
			name: prefix + "nomatch",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: missingFilter,
				Sorter: &boruta.SortInfo{},
			})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
		&testCase{
			// Bad sort item.
			name: prefix + "bad-sorter-item",
			path: path,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: validFilter,
				Sorter: sorterBad,
			})),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)
	sorter := new(boruta.SortInfo)

	// list some
	list, err := client.ListWorkers(validFilter, sorter)
	assert.Nil(err)
	assert.Equal(workers[:2], list)

	// list all
	list, err = client.ListWorkers(nil, nil)
	assert.Nil(err)
	assert.Equal(workers, list)

	// no matches
	list, err = client.ListWorkers(missingFilter, sorter)
	assert.Nil(err)
	assert.Empty(list)

	// list all, sorted by UUID (ascending)
	sorter.Item = "UUID"
	list, err = client.ListWorkers(nil, sorter)
	assert.Nil(err)
	assert.Equal(workers, list)

	// list all, sorted by state (descending)
	sorter.Item = "state"
	sorter.Order = boruta.SortOrderDesc
	w2 := []boruta.WorkerInfo{workers[2], workers[0], workers[3], workers[1]}
	list, err = client.ListWorkers(nil, sorter)
	assert.Nil(err)
	assert.Equal(w2, list)

	// Bad sort item.
	sorter.Item = "foobarbaz"
	list, err = client.ListWorkers(validFilter, sorter)
	assert.Empty(list)
	assert.Equal(util.NewServerError(boruta.ErrWrongSortItem), err)

	// Wrong filter type.
	list, err = client.ListWorkers(new(dummyListFilter), sorter)
	assert.Empty(list)
	assert.Equal(errors.New("only *filter.Workers type is supported"), err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	list, err = client.ListWorkers(nil, nil)
	assert.Zero(list)
	assert.NotNil(err)
}

func TestGetWorkerInfo(t *testing.T) {
	prefix := "worker-info-"
	path := "/api/v1/workers/"
	worker := newWorker(validUUID, boruta.IDLE, boruta.Groups{}, nil)
	header := make(http.Header)
	header.Set(util.WorkerStateHdr, "IDLE")

	tests := []*testCase{
		&testCase{
			// valid
			name:        prefix + "valid",
			path:        path + validUUID,
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      header,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID,
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodGet, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	info, err := client.GetWorkerInfo(validUUID)
	assert.Nil(err)
	assert.Equal(worker, info)

	// invalid UUID
	info, err = client.GetWorkerInfo(invalidID)
	assert.Zero(info)
	assert.Equal(util.NewServerError(util.ErrBadUUID), err)

	// http.Get failure
	client.url = "http://nosuchaddress.fail"
	info, err = client.GetWorkerInfo(validUUID)
	assert.Zero(info)
	assert.NotNil(err)
}

func TestSetState(t *testing.T) {
	prefix := "worker-set-state-"
	path := "/api/v1/workers/"

	tests := []*testCase{
		&testCase{
			// valid
			name:        prefix + "valid",
			path:        path + validUUID + "/setstate",
			json:        string(jsonMustMarshal(&util.WorkerStatePack{WorkerState: boruta.IDLE})),
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID + "/setstate",
			json:        string(jsonMustMarshal(&util.WorkerStatePack{WorkerState: boruta.FAIL})),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	assert.Nil(client.SetState(validUUID, boruta.IDLE))

	// invalid UUID
	assert.Equal(util.NewServerError(util.ErrBadUUID), client.SetState(invalidID, boruta.FAIL))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.SetState(validUUID, boruta.FAIL))
}

func TestSetGroups(t *testing.T) {
	prefix := "worker-set-groups-"
	path := "/api/v1/workers/"
	groups := boruta.Groups{"foo", "bar"}

	tests := []*testCase{
		&testCase{
			// valid
			name:        prefix + "valid",
			path:        path + validUUID + "/setgroups",
			json:        string(jsonMustMarshal(groups)),
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID + "/setgroups",
			json:        string(jsonMustMarshal(groups)),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	assert.Nil(client.SetGroups(validUUID, groups))

	// invalid UUID
	assert.Equal(util.NewServerError(util.ErrBadUUID), client.SetGroups(invalidID, groups))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.SetGroups(validUUID, groups))
}

func TestDeregister(t *testing.T) {
	prefix := "worker-deregister-"
	path := "/api/v1/workers/"

	tests := []*testCase{
		&testCase{
			// valid
			name:        prefix + "valid",
			path:        path + validUUID + "/deregister",
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID + "/deregister",
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	assert.Nil(client.Deregister(validUUID))

	// invalid UUID
	assert.Equal(util.NewServerError(util.ErrBadUUID), client.Deregister(invalidID))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.Deregister(validUUID))
}

func TestGetRequestState(t *testing.T) {
	prefix := "get-request-state-"
	path := "/api/v1/reqs/"

	header := make(http.Header)
	header.Set(util.RequestStateHdr, string(boruta.DONE))

	tests := []*testCase{
		&testCase{
			// valid request
			name:   prefix + "valid",
			path:   path + "1",
			status: http.StatusNoContent,
			header: header,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "2",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
	}

	srv := prepareServer(http.MethodHead, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid request
	state, err := client.GetRequestState(boruta.ReqID(1))
	assert.Nil(err)
	assert.Equal(boruta.DONE, state)

	// missing request
	state, err = client.GetRequestState(boruta.ReqID(2))
	assert.Equal(errors.New("bad HTTP status: 404 Not Found"), err)
}

func TestGetWorkerState(t *testing.T) {
	prefix := "get-worker-state-"
	path := "/api/v1/workers/"

	header := make(http.Header)
	header.Set(util.WorkerStateHdr, string(boruta.RUN))

	tests := []*testCase{
		&testCase{
			// valid
			name:   prefix + "valid",
			path:   path + validUUID,
			status: http.StatusNoContent,
			header: header,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID,
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodHead, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	state, err := client.GetWorkerState(validUUID)
	assert.Nil(err)
	assert.Equal(boruta.RUN, state)

	// invalid UUID
	state, err = client.GetWorkerState(invalidID)
	assert.Equal(errors.New("bad HTTP status: 400 Bad Request"), err)
}

func TestGetJobTimeout(t *testing.T) {
	prefix := "get-job-timeout-"
	path := "/api/v1/reqs/"
	date := time.Now().Round(time.Second)

	header := make(http.Header)
	header.Set(util.RequestStateHdr, string(boruta.INPROGRESS))
	header.Set(util.JobTimeoutHdr, date.Format(util.DateFormat))

	wait := make(http.Header)
	wait.Set(util.RequestStateHdr, string(boruta.WAIT))

	bad := make(http.Header)
	bad.Set(util.RequestStateHdr, string(boruta.INPROGRESS))
	bad.Set(util.JobTimeoutHdr, "fail")

	tests := []*testCase{
		&testCase{
			// valid request
			name:   prefix + "valid",
			path:   path + "1",
			status: http.StatusNoContent,
			header: header,
		},
		&testCase{
			// request in wrong state
			name:   prefix + "wrong-state",
			path:   path + "2",
			status: http.StatusNoContent,
			header: wait,
		},
		&testCase{
			// missing request
			name:        prefix + "missing",
			path:        path + "3",
			contentType: contentJSON,
			status:      http.StatusNotFound,
		},
		&testCase{
			// invalid date format
			name:        prefix + "bad-date",
			path:        path + "4",
			contentType: contentJSON,
			status:      http.StatusNoContent,
			header:      bad,
		},
	}

	srv := prepareServer(http.MethodHead, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	timeout, err := client.GetJobTimeout(boruta.ReqID(1))
	assert.Nil(err)
	assert.True(date.Equal(timeout))

	// wrong state
	_, err = client.GetJobTimeout(boruta.ReqID(2))
	assert.Equal(errors.New(`request must be in "IN PROGRESS" state`), err)

	// missing
	_, err = client.GetJobTimeout(boruta.ReqID(3))
	assert.Equal(errors.New("bad HTTP status: 404 Not Found"), err)

	// wrong date format
	_, err = client.GetJobTimeout(boruta.ReqID(4))
	assert.NotNil(err)
	var parseErr *time.ParseError
	assert.IsType(parseErr, err)
}

func TestVersion(t *testing.T) {
	name := "get-api-version"
	path := "/api/v1/version"

	valid := &Version{
		Client: boruta.Version,
		BorutaVersion: util.BorutaVersion{
			Server: boruta.Version,
			API:    "v1",
			State:  util.Devel,
		},
	}

	header := make(http.Header)
	header.Set(util.ServerVersionHdr, valid.Server)
	header.Set(util.APIVersionHdr, valid.API)
	header.Set(util.APIStateHdr, valid.State)

	tests := []*testCase{
		&testCase{
			// valid request
			name:   name,
			path:   path,
			status: http.StatusNoContent,
			header: header,
		},
	}

	srv := prepareServer(http.MethodHead, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	version, err := client.Version()
	assert.Nil(err)
	assert.Equal(valid, version)

	client.url = "http://nosuchaddress.fail"
	version, err = client.Version()
	assert.Nil(version)
	assert.NotNil(err)
}
