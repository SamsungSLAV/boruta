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
	// response that should be returned (read from server testdata if empty)
	resp string
	// expected headers
	header http.Header
}

// chanFilter is used to mock failure of JSON marshalling in ListRequests.
type chanFilter chan bool

func (f chanFilter) Match(_ *ReqInfo) bool {
	return false
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
	invalidID   = "test"
	validUUID   = "ec4898ac-0853-407c-8501-cbb24ef6bd77"
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
		w.WriteHeader(test.status)
		if test.status != http.StatusNoContent {
			w.Write(data)
		}
	}

	mux.Handle(apiPrefix, http.HandlerFunc(handler))
	return httptest.NewServer(mux)
}

// from http/server/api/v1/api_test.go
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
	req.ID = ReqID(1)
	req.State = WAIT
	reqInfo, err := client.GetRequestInfo(ReqID(1))
	assert.Nil(err)
	assert.Equal(req, reqInfo)

	// missing
	reqInfo, err = client.GetRequestInfo(ReqID(2))
	assert.Zero(reqInfo)
	assert.Equal(errReqNotFound, err)

	// http.Get failure
	client.url = "http://nosuchaddress.fail"
	reqInfo, err = client.GetRequestInfo(ReqID(1))
	assert.Zero(reqInfo)
	assert.NotNil(err)
}

func TestListRequests(t *testing.T) {
	prefix := "filter-reqs-"
	path := "/api/v1/reqs/list"

	// from api/v1 TestListRequestsHandler
	deadline, _ := time.Parse(dateLayout, "2222-12-31T00:00:00Z")
	validAfter, _ := time.Parse(dateLayout, "1683-09-12T00:00:00Z")
	reqs := []ReqInfo{
		{ID: 1, Priority: (HiPrio + LoPrio) / 2, State: WAIT,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 2, Priority: (HiPrio+LoPrio)/2 + 1, State: WAIT,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 3, Priority: (HiPrio + LoPrio) / 2, State: CANCEL,
			Deadline: deadline, ValidAfter: validAfter},
		{ID: 4, Priority: (HiPrio+LoPrio)/2 + 1, State: CANCEL,
			Deadline: deadline, ValidAfter: validAfter},
	}

	missingFilter := util.NewRequestFilter("INPROGRESS", "2")
	missingHeader := make(http.Header)
	missingHeader.Set("Boruta-Request-Count", "0")
	validFilter := util.NewRequestFilter("WAIT", "")
	validHeader := make(http.Header)
	validHeader.Set("Boruta-Request-Count", "2")
	nilHeader := make(http.Header)
	nilHeader.Set("Boruta-Request-Count", "4")

	tests := []*testCase{
		&testCase{
			// valid filter
			name:        prefix + "valid-filter",
			path:        path,
			json:        string(jsonMustMarshal(validFilter)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      validHeader,
		},
		&testCase{
			// nil filter - list all
			name:        prefix + "nil",
			path:        path,
			json:        string(jsonMustMarshal(nil)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      nilHeader,
		},
		&testCase{
			// no requests matched
			name:        prefix + "nomatch-all",
			path:        path,
			json:        string(jsonMustMarshal(missingFilter)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// nil filter
	list, err := client.ListRequests(nil)
	assert.Nil(err)
	assert.Equal(reqs, list)

	// valid filter
	list, err = client.ListRequests(validFilter)
	assert.Nil(err)
	assert.Equal(reqs[:2], list)

	// no requests matched
	list, err = client.ListRequests(missingFilter)
	assert.Nil(err)
	assert.Equal([]ReqInfo{}, list)

	// json.Marshal failure
	var typeError *json.UnsupportedTypeError
	list, err = client.ListRequests(make(chanFilter))
	assert.Empty(list)
	assert.IsType(typeError, err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	list, err = client.ListRequests(nil)
	assert.Zero(list)
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

	access := AccessInfo{
		Addr: &net.TCPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 22,
		},
		Key:      *key,
		Username: "wołchw",
	}

	// valid
	accessInfo, err := client.AcquireWorker(ReqID(1))
	assert.Equal(access, accessInfo)
	assert.Nil(err)

	// missing
	accessInfo, err = client.AcquireWorker(ReqID(2))
	assert.Zero(accessInfo)
	assert.Equal(errReqNotFound, err)

	// bad key
	accessInfo, err = client.AcquireWorker(ReqID(3))
	assert.Zero(accessInfo)
	assert.Equal(errors.New("wrong key: "+badkeyAI.Key), err)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	accessInfo, err = client.AcquireWorker(ReqID(1))
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
	assert.Nil(client.ProlongAccess(ReqID(1)))

	// missing
	assert.Equal(errReqNotFound, client.ProlongAccess(ReqID(2)))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.ProlongAccess(ReqID(1)))
}

func TestListWorkers(t *testing.T) {
	prefix := "filter-workers-"
	path := "/api/v1/workers/list"

	// based on http/server/api/v1/handlers_test.go
	armCaps := make(Capabilities)
	armCaps["architecture"] = "AArch64"
	riscvCaps := make(Capabilities)
	riscvCaps["architecture"] = "RISC-V"
	workers := []WorkerInfo{
		newWorker("0", IDLE, Groups{"Lędzianie"}, armCaps),
		newWorker("1", FAIL, Groups{"Malinowy Chruśniak"}, armCaps),
		newWorker("2", IDLE, Groups{"Malinowy Chruśniak", "Lędzianie"}, riscvCaps),
		newWorker("3", FAIL, Groups{"Malinowy Chruśniak"}, riscvCaps),
	}
	validFilter := util.WorkersFilter{
		Groups:       Groups{"Lędzianie"},
		Capabilities: map[string]string{"architecture": "AArch64"},
	}
	validHeader := make(http.Header)
	validHeader.Set("Boruta-Worker-Count", "2")
	allHeader := make(http.Header)
	allHeader.Set("Boruta-Worker-Count", "4")
	missingFilter := util.WorkersFilter{
		Groups: Groups{"Fern Flower"},
	}
	missingHeader := make(http.Header)
	missingHeader.Set("Boruta-Worker-Count", "0")

	tests := []*testCase{
		&testCase{
			// valid request
			name:        prefix + "valid-filter",
			path:        path,
			json:        string(jsonMustMarshal(validFilter)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      validHeader,
		},
		&testCase{
			// list all
			name:        prefix + "empty-filter",
			path:        path,
			json:        string(jsonMustMarshal(util.WorkersFilter{nil, nil})),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		&testCase{
			// no matches
			name:        prefix + "nomatch",
			path:        path,
			json:        string(jsonMustMarshal(missingFilter)),
			contentType: contentJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// list some
	list, err := client.ListWorkers(validFilter.Groups, validFilter.Capabilities)
	assert.Nil(err)
	assert.Equal(workers[:2], list)

	// list all
	list, err = client.ListWorkers(nil, nil)
	assert.Nil(err)
	assert.Equal(workers, list)

	// no matches
	list, err = client.ListWorkers(missingFilter.Groups, missingFilter.Capabilities)
	assert.Nil(err)
	assert.Empty(list)

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	list, err = client.ListWorkers(nil, nil)
	assert.Zero(list)
	assert.NotNil(err)
}

func TestGetWorkerInfo(t *testing.T) {
	prefix := "worker-info-"
	path := "/api/v1/workers/"
	worker := newWorker(validUUID, IDLE, Groups{}, nil)
	header := make(http.Header)
	header.Set("Boruta-Worker-State", "IDLE")

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
			json:        string(jsonMustMarshal(&util.WorkerStatePack{IDLE})),
			contentType: contentJSON,
			status:      http.StatusNoContent,
		},
		&testCase{
			// invalid UUID
			name:        prefix + "bad-uuid",
			path:        path + invalidID + "/setstate",
			json:        string(jsonMustMarshal(&util.WorkerStatePack{FAIL})),
			contentType: contentJSON,
			status:      http.StatusBadRequest,
		},
	}

	srv := prepareServer(http.MethodPost, tests)
	defer srv.Close()
	assert, client := initTest(t, srv.URL)

	// valid
	assert.Nil(client.SetState(validUUID, IDLE))

	// invalid UUID
	assert.Equal(util.NewServerError(util.ErrBadUUID), client.SetState(invalidID, FAIL))

	// http.Post failure
	client.url = "http://nosuchaddress.fail"
	assert.NotNil(client.SetState(validUUID, FAIL))
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
