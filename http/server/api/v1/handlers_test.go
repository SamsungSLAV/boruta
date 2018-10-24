/* *  Copyright (c) 2017-2018 Samsung Electronics Co., Ltd All Rights Reserved
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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	util "github.com/SamsungSLAV/boruta/http"
	"github.com/SamsungSLAV/boruta/requests"
)

func TestNewRequestHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	prefix := "new-req-"
	path := "/api/v1/reqs/"
	methods := []string{http.MethodPost}
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, path, methods...)

	var req boruta.ReqInfo
	err := json.Unmarshal([]byte(validReqJSON), &req)
	assert.Nil(err)

	m.rq.EXPECT().NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter,
		req.Deadline).Return(boruta.ReqID(1), nil)
	m.rq.EXPECT().NewRequest(req.Caps, boruta.Priority(32), req.Owner, req.ValidAfter,
		req.Deadline).Return(boruta.ReqID(0), requests.ErrPriority)

	tests := []requestTest{
		{
			// valid request
			name:        prefix + "valid",
			path:        path,
			methods:     methods,
			json:        validReqJSON,
			contentType: contentTypeJSON,
			status:      http.StatusCreated,
		},
		{
			// bad request - priority out of bounds
			name:        prefix + "bad-prio",
			path:        path,
			methods:     methods,
			json:        `{"Priority":32,"Deadline":"2200-12-31T01:02:03Z","ValidAfter":"2100-01-01T04:05:06Z","Caps":{"architecture":"armv7l","monitor":"yes"}}`,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		// bad request - malformed request JSON
		malformedJSONTest,
	}

	runTests(assert, r, tests)
}

func TestCloseRequestHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	pathfmt := "/api/v1/reqs/%s/close"
	prefix := "close-req-"

	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, fmt.Sprintf(pathfmt, invalidID), methods...)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().CloseRequest(boruta.ReqID(1)).Return(nil)
	m.rq.EXPECT().CloseRequest(boruta.ReqID(2)).Return(boruta.NotFoundError("Request"))

	tests := []requestTest{
		// Close valid request in state WAIT (cancel).
		{
			name:        prefix + "valid",
			path:        fmt.Sprintf(pathfmt, "1"),
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		// Try to close request with invalid ID.
		invalidIDTest,
		// Try to close request which doesn't exist.
		notFoundTest,
	}

	runTests(assert, r, tests)
}

func TestUpdateRequestHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	path := "/api/v1/reqs/"
	setPrioJSON := `{"Priority":4}`
	prefix := "update-req-"

	valid := &boruta.ReqInfo{
		ID:       boruta.ReqID(1),
		Priority: boruta.Priority(4),
	}
	missing := &boruta.ReqInfo{
		ID:       boruta.ReqID(2),
		Priority: boruta.Priority(4),
	}
	m.rq.EXPECT().UpdateRequest(valid).Return(nil).Times(2)
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, path+"1", methods...)
	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, path+invalidID, methods...)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, path+"2", methods...)
	notFoundTest.json = setPrioJSON
	m.rq.EXPECT().UpdateRequest(missing).Return(boruta.NotFoundError("Request"))

	var req boruta.ReqInfo
	var err error
	req.Deadline, err = time.Parse(dateLayout, future)
	assert.Nil(err)
	req.ValidAfter, err = time.Parse(dateLayout, past)
	assert.Nil(err)

	tests := []requestTest{
		{
			// valid - only priority in JSON
			name:        prefix + "prio",
			path:        path + "1",
			methods:     methods,
			json:        setPrioJSON,
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			// valid - full JSON
			name:        prefix + "valid",
			path:        path + "1",
			methods:     methods,
			json:        string(jsonMustMarshal(valid)),
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			// no action
			name:        prefix + "empty",
			path:        path + "1",
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		{
			// Request ID mismatch between URL and JSON
			name:        prefix + "mismatch",
			path:        path + "2",
			methods:     methods,
			json:        string(jsonMustMarshal(valid)),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		malformedJSONTest,
		invalidIDTest,
		notFoundTest,
	}

	runTests(assert, r, tests)
}

func TestGetRequestInfoHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	methods := []string{http.MethodGet, http.MethodHead}
	prefix := "req-info-"
	path := "/api/v1/reqs/"

	var req boruta.ReqInfo
	err := json.Unmarshal([]byte(validReqJSON), &req)
	assert.Nil(err)
	header := make(http.Header)
	header.Set("Boruta-Request-State", "WAITING")

	timeout, err := time.Parse(dateLayout, future)
	assert.Nil(err)
	var running boruta.ReqInfo
	err = json.Unmarshal([]byte(validReqJSON), &running)
	assert.Nil(err)
	running.ID = boruta.ReqID(2)
	running.State = boruta.INPROGRESS
	running.Job = &boruta.JobInfo{
		WorkerUUID: validUUID,
		Timeout:    timeout,
	}
	rheader := make(http.Header)
	rheader.Set("Boruta-Request-State", string(boruta.INPROGRESS))
	rheader.Set("Boruta-Job-Timeout", timeout.Format(util.DateFormat))

	notFoundTest := testFromTempl(notFoundTestTempl, prefix, path+"3", methods...)
	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, path+invalidID, methods...)
	m.rq.EXPECT().GetRequestInfo(boruta.ReqID(1)).Return(req, nil).Times(2)
	m.rq.EXPECT().GetRequestInfo(boruta.ReqID(2)).Return(running, nil).Times(2)
	m.rq.EXPECT().GetRequestInfo(boruta.ReqID(3)).Return(boruta.ReqInfo{}, boruta.NotFoundError("Request")).Times(2)

	tests := []requestTest{
		// Get information of existing request.
		{
			name:        "req-info",
			path:        path + "1",
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      header,
		},
		// Get information of running request.
		{
			name:        "req-info-running",
			path:        path + "2",
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      rheader,
		},
		// Try to get request information of request that doesn't exist.
		notFoundTest,
		// Try to get request with invalid ID.
		invalidIDTest,
	}

	runTests(assert, r, tests)
}

func TestListRequestsHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	deadline, err := time.Parse(dateLayout, future)
	assert.Nil(err)
	validAfter, err := time.Parse(dateLayout, past)
	assert.Nil(err)
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

	methods := []string{http.MethodPost}
	prefix := "filter-reqs-"
	filterPath := "/api/v1/reqs/list"
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, filterPath, methods...)

	validFilter := filter.NewRequests("WAIT", "")
	m.rq.EXPECT().ListRequests(validFilter).Return(reqs[:2], nil)
	validHeader := make(http.Header)
	validHeader.Set("Boruta-Request-Count", "2")

	emptyFilter := filter.NewRequests("", "")
	m.rq.EXPECT().ListRequests(emptyFilter).Return(reqs, nil).Times(2)
	m.rq.EXPECT().ListRequests(nil).Return(reqs, nil).Times(3)
	allHeader := make(http.Header)
	allHeader.Set("Boruta-Request-Count", "4")

	missingFilter := filter.NewRequests("INVALID", "")
	m.rq.EXPECT().ListRequests(missingFilter).Return([]boruta.ReqInfo{}, nil)
	missingHeader := make(http.Header)
	missingHeader.Set("Boruta-Request-Count", "0")

	// Currently ListRequests doesn't return any error hence the meaningless values.
	badFilter := filter.NewRequests("FAIL", "-1")
	m.rq.EXPECT().ListRequests(badFilter).Return([]boruta.ReqInfo{}, errors.New("foo bar: pizza failed"))

	tests := []requestTest{
		// Valid filter - list some requests.
		{
			name:        prefix + "valid-filter",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(validFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      validHeader,
		},
		// List all requests.
		{
			name:        "list-reqs-all",
			path:        "/api/v1/reqs/",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty body - list all requests.
		{
			name:        prefix + "empty-json",
			path:        filterPath,
			methods:     methods,
			json:        "",
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Nil filter - list all requests (same as emptyFilter).
		{
			name:        prefix + "nil",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(nil)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty filter - list all requests.
		{
			name:        prefix + "empty",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(emptyFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// No matches
		{
			name:        prefix + "nomatch-all",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(missingFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
		// Error
		{
			name:        prefix + "bad-filter",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(badFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		malformedJSONTest,
	}

	runTests(assert, r, tests)
}

func TestAcquireWorkerHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	prefix := "acquire-worker-"
	pathfmt := "/api/v1/reqs/%s/acquire_worker"

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

	m.rq.EXPECT().AcquireWorker(boruta.ReqID(1)).Return(access, nil)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().AcquireWorker(boruta.ReqID(2)).Return(boruta.AccessInfo{}, boruta.NotFoundError("Request"))
	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, fmt.Sprintf(pathfmt, invalidID), methods...)

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        fmt.Sprintf(pathfmt, "1"),
			methods:     methods,
			json:        string(jsonMustMarshal(access)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		notFoundTest,
		invalidIDTest,
	}

	runTests(assert, r, tests)
}

func TestProlongAccessHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	prefix := "prolong-access-"
	pathfmt := "/api/v1/reqs/%s/prolong"

	m.rq.EXPECT().ProlongAccess(boruta.ReqID(1)).Return(nil)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().ProlongAccess(boruta.ReqID(2)).Return(boruta.NotFoundError("Request"))
	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, fmt.Sprintf(pathfmt, invalidID), methods...)

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        fmt.Sprintf(pathfmt, "1"),
			methods:     methods,
			json:        "",
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		notFoundTest,
		invalidIDTest,
	}

	runTests(assert, r, tests)
}

func TestListWorkersHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

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

	methods := []string{http.MethodPost}
	prefix := "filter-workers-"
	filterPath := "/api/v1/workers/list"
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, filterPath, methods...)

	validFilter := filter.Workers{
		Groups:       boruta.Groups{"Lędzianie"},
		Capabilities: map[string]string{"architecture": "AArch64"},
	}
	m.wm.EXPECT().ListWorkers(validFilter.Groups, validFilter.Capabilities).Return(workers[:2], nil)
	validHeader := make(http.Header)
	validHeader.Set("Boruta-Worker-Count", "2")

	m.wm.EXPECT().ListWorkers(nil, nil).Return(workers, nil).MinTimes(1)
	m.wm.EXPECT().ListWorkers(boruta.Groups{}, nil).Return(workers, nil)
	m.wm.EXPECT().ListWorkers(nil, make(boruta.Capabilities)).Return(workers, nil)
	allHeader := make(http.Header)
	allHeader.Set("Boruta-Worker-Count", "4")

	missingFilter := filter.Workers{
		Groups: boruta.Groups{"Fern Flower"},
	}
	missingHeader := make(http.Header)
	missingHeader.Set("Boruta-Worker-Count", "0")
	m.wm.EXPECT().ListWorkers(missingFilter.Groups, missingFilter.Capabilities).Return([]boruta.WorkerInfo{}, nil)

	// Currently ListWorkers doesn't return any error hence the meaningless values.
	badFilter := filter.Workers{
		Groups: boruta.Groups{"Oops"},
	}
	m.wm.EXPECT().ListWorkers(badFilter.Groups, badFilter.Capabilities).Return([]boruta.WorkerInfo{}, errors.New("foo bar: pizza failed"))

	tests := []requestTest{
		// Valid filter - list some requests.
		{
			name:        prefix + "valid-filter",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(validFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      validHeader,
		},
		// List all requests.
		{
			name:        "list-workers-all",
			path:        "/api/v1/workers/",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty body - list all requests.
		{
			name:        prefix + "empty-body",
			path:        filterPath,
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty filter (all nil) - list all requests.
		{
			name:    prefix + "empty-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(filter.Workers{
				Groups:       nil,
				Capabilities: nil})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty filter (nil groups) - list all requests.
		{
			name:    prefix + "empty2-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(filter.Workers{
				Groups:       nil,
				Capabilities: make(boruta.Capabilities)})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// Empty filter (nil caps) - list all requests.
		{
			name:    prefix + "empty3-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(filter.Workers{
				Groups:       boruta.Groups{},
				Capabilities: nil})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      allHeader,
		},
		// No matches.
		{
			name:        prefix + "nomatch",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(missingFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      missingHeader,
		},
		// Error.
		{
			name:        prefix + "bad-filter",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(badFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		malformedJSONTest,
	}

	runTests(assert, r, tests)
}

func TestGetWorkerInfoHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	prefix := "worker-info-"
	path := "/api/v1/workers/"
	methods := []string{http.MethodGet, http.MethodHead}
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, path+missingUUID, methods...)

	worker := newWorker(validUUID, boruta.IDLE, boruta.Groups{}, nil)
	missingErr := boruta.NotFoundError("Worker")
	m.wm.EXPECT().GetWorkerInfo(boruta.WorkerUUID(validUUID)).Return(worker, nil).Times(2)
	m.wm.EXPECT().GetWorkerInfo(boruta.WorkerUUID(missingUUID)).Return(boruta.WorkerInfo{}, missingErr).Times(2)
	header := make(http.Header)
	header.Set("Boruta-Worker-State", "IDLE")

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        path + validUUID,
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      header,
		},
		{
			name:        prefix + "bad-uuid",
			path:        path + invalidID,
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		notFoundTest,
	}

	runTests(assert, r, tests)
}

func TestSetWorkerStateHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	prefix := "worker-set-state-"
	path := "/api/v1/workers/%s/setstate"
	methods := []string{http.MethodPost}

	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(path, missingUUID), methods...)
	notFoundTest.json = string(jsonMustMarshal(util.WorkerStatePack{WorkerState: boruta.IDLE}))
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, fmt.Sprintf(path, validUUID), methods...)
	missingErr := boruta.NotFoundError("Worker")

	m.wm.EXPECT().SetState(boruta.WorkerUUID(validUUID), boruta.IDLE).Return(nil)
	m.wm.EXPECT().SetState(boruta.WorkerUUID(missingUUID), boruta.IDLE).Return(missingErr)

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        fmt.Sprintf(path, validUUID),
			methods:     methods,
			json:        string(jsonMustMarshal(util.WorkerStatePack{WorkerState: boruta.IDLE})),
			contentType: contentTypeJSON,
			status:      http.StatusAccepted,
		},
		{
			name:        prefix + "bad-uuid",
			path:        fmt.Sprintf(path, invalidID),
			methods:     methods,
			json:        string(jsonMustMarshal(util.WorkerStatePack{WorkerState: boruta.IDLE})),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		notFoundTest,
		malformedJSONTest,
	}

	runTests(assert, r, tests)
}

func TestSetWorkerGroupsHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	path := "/api/v1/workers/%s/setgroups"
	methods := []string{http.MethodPost}
	prefix := "worker-set-groups-"

	groups := boruta.Groups{"foo", "bar"}
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(path, missingUUID), methods...)
	notFoundTest.json = string(jsonMustMarshal(groups))
	missingErr := boruta.NotFoundError("Worker")
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, fmt.Sprintf(path, validUUID), methods...)

	m.wm.EXPECT().SetGroups(boruta.WorkerUUID(validUUID), groups).Return(nil)
	m.wm.EXPECT().SetGroups(boruta.WorkerUUID(missingUUID), groups).Return(missingErr)

	tests := []requestTest{
		// Set valid groups.
		{
			name:    prefix + "valid",
			path:    fmt.Sprintf(path, validUUID),
			methods: methods,
			json:    string(jsonMustMarshal(groups)),
			status:  http.StatusNoContent,
		},
		// invalid UUID
		{
			name:        prefix + "bad-uuid",
			path:        fmt.Sprintf(path, invalidID),
			methods:     methods,
			json:        string(jsonMustMarshal(groups)),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		notFoundTest,
		malformedJSONTest,
	}

	runTests(assert, r, tests)
}

func TestDeregisterWorkerHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()

	prefix := "worker-deregister-"
	methods := []string{http.MethodPost}
	pathfmt := "/api/v1/workers/%s/deregister"

	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, missingUUID), methods...)

	m.wm.EXPECT().Deregister(boruta.WorkerUUID(validUUID)).Return(nil)
	m.wm.EXPECT().Deregister(boruta.WorkerUUID(missingUUID)).Return(boruta.NotFoundError("Worker"))

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        fmt.Sprintf(pathfmt, validUUID),
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNoContent,
		},
		{
			name:        prefix + "bad-uuid",
			path:        fmt.Sprintf(pathfmt, invalidID),
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		notFoundTest,
	}

	runTests(assert, r, tests)
}

func TestVersionHandler(t *testing.T) {
	assert, m, r := initTest(t)
	defer m.finish()
	header := make(http.Header)
	header.Set("Boruta-Server-Version", boruta.Version)
	header.Set("Boruta-API-Version", Version)
	header.Set("Boruta-API-State", State)

	tests := []requestTest{
		{
			name:        "api-version",
			path:        "/api/v1/version",
			methods:     []string{http.MethodGet, http.MethodGet},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
			header:      header,
		},
	}

	runTests(assert, r, tests)
}
