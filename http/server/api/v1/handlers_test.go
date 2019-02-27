/* *  Copyright (c) 2017-2019 Samsung Electronics Co., Ltd All Rights Reserved
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

var (
	defaultSorter = &boruta.SortInfo{} // "", SortOrderAsc

	sorterBad = &boruta.SortInfo{
		Item:  "foobarbaz",
		Order: boruta.SortOrderAsc,
	}
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
		},
		// Get information of running request.
		{
			name:        "req-info-running",
			path:        path + "2",
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
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

	sorterAsc := &boruta.SortInfo{
		Item:  "priority",
		Order: boruta.SortOrderAsc,
	}

	sorterDesc := &boruta.SortInfo{
		Item:  "state",
		Order: boruta.SortOrderDesc,
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

	methods := []string{http.MethodPost}
	prefix := "filter-reqs-"
	filterPath := "/api/v1/reqs/list"
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, filterPath, methods...)

	validFilter := filter.NewRequests(nil, nil, []boruta.ReqState{boruta.WAIT})
	m.rq.EXPECT().ListRequests(validFilter, defaultSorter, nil).Return(reqs[:2], nil, nil)
	m.rq.EXPECT().ListRequests(validFilter, sorterAsc, nil).Return(reqs[:2], validInfo, nil)
	m.rq.EXPECT().ListRequests(validFilter, sorterDesc, nil).Return([]boruta.ReqInfo{reqs[1],
		reqs[0]}, validInfo, nil)
	m.rq.EXPECT().ListRequests(validFilter, sorterBad, nil).Return([]boruta.ReqInfo{}, nil,
		boruta.ErrWrongSortItem)

	emptyFilter := filter.NewRequests(nil, nil, nil)
	m.rq.EXPECT().ListRequests(emptyFilter, nil, nil).Return(reqs, secondInfo, nil)
	m.rq.EXPECT().ListRequests(emptyFilter, nil, fwdPaginator1).Return(reqs[:2], firstInfo, nil)
	m.rq.EXPECT().ListRequests(emptyFilter, nil, fwdPaginator2).Return(reqs[2:], secondInfo, nil)
	m.rq.EXPECT().ListRequests(emptyFilter, nil, backPaginator1).Return(reqs[2:], firstInfo, nil)
	m.rq.EXPECT().ListRequests(emptyFilter, nil, backPaginator2).Return(reqs[:2], secondInfo, nil)
	m.rq.EXPECT().ListRequests(nil, nil, nil).Return(reqs, secondInfo, nil).Times(2)

	missingFilter := filter.NewRequests(nil, nil, []boruta.ReqState{boruta.INVALID})
	m.rq.EXPECT().ListRequests(missingFilter, nil, nil).Return([]boruta.ReqInfo{}, nil, nil)

	// Currently ListRequests doesn't return any error hence the meaningless values.
	badFilter := filter.NewRequests(nil, []boruta.Priority{99}, []boruta.ReqState{"NoSuchState"})
	m.rq.EXPECT().ListRequests(badFilter, nil, nil).Return([]boruta.ReqInfo{}, nil,
		errors.New("foo bar: pizza failed"))

	tests := []requestTest{
		// Valid filter - list some requests.
		{
			name:    prefix + "valid-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: validFilter,
				Sorter: defaultSorter,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Valid filter with sorter - list some requests sorted by priority (ascending).
		{
			name:    prefix + "valid-filter-sort-asc",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterAsc,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Valid filter with sorter - list some requests sorted by state (descending).
		// As state is equal - this will be sorted by ID.
		{
			name:    prefix + "valid-filter-sort-desc",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterDesc,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty body - list all requests.
		{
			name:        prefix + "empty-json",
			path:        filterPath,
			methods:     methods,
			json:        "",
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Nil filter - list all requests (same as emptyFilter).
		{
			name:        prefix + "nil",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(nil)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter - list all requests.
		{
			name:    prefix + "empty",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: emptyFilter,
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List first part of all requests.
		{
			name:    prefix + "paginator-fwd1",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter:    emptyFilter,
				Sorter:    nil,
				Paginator: fwdPaginator1,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List second part of all requests.
		{
			name:    prefix + "paginator-fwd2",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter:    emptyFilter,
				Sorter:    nil,
				Paginator: fwdPaginator2,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List first part of all requests (backward).
		{
			name:    prefix + "paginator-back1",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter:    emptyFilter,
				Sorter:    nil,
				Paginator: backPaginator1,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List second part of all requests (backward).
		{
			name:    prefix + "paginator-back2",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter:    emptyFilter,
				Sorter:    nil,
				Paginator: backPaginator2,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// No matches
		{
			name:    prefix + "nomatch-all",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: missingFilter,
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Error in filter.
		{
			name:    prefix + "bad-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: badFilter,
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		// Bad sort item.
		{
			name:    prefix + "bad-sorter-item",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.RequestsListSpec{
				Filter: validFilter,
				Sorter: sorterBad,
			})),
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

	sorterAsc := &boruta.SortInfo{
		Item:  "uuid",
		Order: boruta.SortOrderAsc,
	}

	sorterDesc := &boruta.SortInfo{
		Item:  "state",
		Order: boruta.SortOrderDesc,
	}

	validInfo := &boruta.ListInfo{TotalItems: 2, RemainingItems: 0}
	allInfo := &boruta.ListInfo{TotalItems: 4, RemainingItems: 0}
	emptyInfo := &boruta.ListInfo{TotalItems: 0, RemainingItems: 0}
	firstInfo := &boruta.ListInfo{TotalItems: 4, RemainingItems: 2}
	secondInfo := &boruta.ListInfo{TotalItems: 4, RemainingItems: 0}

	fwdPaginator1 := &boruta.WorkersPaginator{
		ID:        "0",
		Direction: boruta.DirectionForward,
		Limit:     2,
	}
	fwdPaginator2 := &boruta.WorkersPaginator{
		ID:        "2",
		Direction: boruta.DirectionForward,
		Limit:     2,
	}
	backPaginator1 := &boruta.WorkersPaginator{
		ID:        "4",
		Direction: boruta.DirectionBackward,
		Limit:     2,
	}
	backPaginator2 := &boruta.WorkersPaginator{
		ID:        "2",
		Direction: boruta.DirectionBackward,
		Limit:     2,
	}

	workers := []boruta.WorkerInfo{
		newWorker("0", boruta.IDLE, boruta.Groups{"Lędzianie"}, armCaps),
		newWorker("1", boruta.FAIL, boruta.Groups{"Malinowy Chruśniak"}, armCaps),
		newWorker("2", boruta.IDLE, boruta.Groups{"Malinowy Chruśniak", "Lędzianie"},
			riscvCaps),
		newWorker("3", boruta.FAIL, boruta.Groups{"Malinowy Chruśniak"}, riscvCaps),
	}

	methods := []string{http.MethodPost}
	prefix := "filter-workers-"
	filterPath := "/api/v1/workers/list"
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, filterPath, methods...)

	validFilter := filter.NewWorkers(boruta.Groups{"Malinowy Chruśniak"},
		boruta.Capabilities{"architecture": "RISC-V"})
	m.wm.EXPECT().ListWorkers(validFilter, defaultSorter, nil).Return(workers[2:], validInfo,
		nil)
	m.wm.EXPECT().ListWorkers(validFilter, sorterBad, nil).Return([]boruta.WorkerInfo{}, nil,
		boruta.ErrWrongSortItem)

	m.wm.EXPECT().ListWorkers(nil, nil, nil).Return(workers, allInfo, nil).MinTimes(1)
	w2 := []boruta.WorkerInfo{workers[2], workers[0], workers[3], workers[1]}
	m.wm.EXPECT().ListWorkers(filter.NewWorkers(boruta.Groups{}, nil), nil, nil).Return(workers,
		allInfo, nil)
	m.wm.EXPECT().ListWorkers(filter.NewWorkers(nil, make(boruta.Capabilities)), nil,
		nil).Return(workers, allInfo, nil)
	m.wm.EXPECT().ListWorkers(filter.NewWorkers(nil, nil), nil, nil).Return(workers, allInfo,
		nil)
	m.wm.EXPECT().ListWorkers(nil, sorterAsc, nil).Return(workers, allInfo, nil)
	m.wm.EXPECT().ListWorkers(nil, sorterDesc, nil).Return(w2, allInfo, nil)
	m.wm.EXPECT().ListWorkers(nil, nil, fwdPaginator1).Return(workers[:2], firstInfo, nil)
	m.wm.EXPECT().ListWorkers(nil, nil, fwdPaginator2).Return(workers[2:], secondInfo, nil)
	m.wm.EXPECT().ListWorkers(nil, nil, backPaginator1).Return(workers[2:], firstInfo, nil)
	m.wm.EXPECT().ListWorkers(nil, nil, backPaginator2).Return(workers[:2], secondInfo, nil)

	missingFilter := filter.NewWorkers(boruta.Groups{"Fern Flower"}, nil)
	m.wm.EXPECT().ListWorkers(missingFilter, nil, nil).Return([]boruta.WorkerInfo{}, emptyInfo,
		nil)

	// Currently ListWorkers doesn't return any error hence the meaningless values.
	badFilter := filter.NewWorkers(boruta.Groups{"Oops"}, nil)
	m.wm.EXPECT().ListWorkers(badFilter, nil, nil).Return([]boruta.WorkerInfo{}, nil,
		errors.New("foo bar: pizza failed"))

	tests := []requestTest{
		// Valid filter - list some workers.
		{
			name:    prefix + "valid-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: validFilter,
				Sorter: defaultSorter,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty body - list all workers.
		{
			name:        prefix + "empty-body",
			path:        filterPath,
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter (all nil) - list all workers.
		{
			name:    prefix + "empty-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: &filter.Workers{Groups: nil, Capabilities: nil},
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter - list all workers sorted by uuid (ascending).
		{
			name:    prefix + "empty-filter-sort-asc",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: nil,
				Sorter: sorterAsc,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter - list all workers sorted by state (descending).
		{
			name:    prefix + "empty-filter-sort-desc",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: nil,
				Sorter: sorterDesc,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter (nil groups) - list all workers.
		{
			name:    prefix + "empty2-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: &filter.Workers{
					Groups:       nil,
					Capabilities: make(boruta.Capabilities),
				},
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Empty filter (nil caps) - list all workers.
		{
			name:    prefix + "empty3-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: &filter.Workers{Groups: boruta.Groups{}, Capabilities: nil},
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List first part of all workers.
		{
			name:    prefix + "paginator-fwd1",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter:    nil,
				Sorter:    nil,
				Paginator: fwdPaginator1,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List second part of all workers.
		{
			name:    prefix + "paginator-fwd2",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter:    nil,
				Sorter:    nil,
				Paginator: fwdPaginator2,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List first part of all workers (backward).
		{
			name:    prefix + "paginator-back1",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter:    nil,
				Sorter:    nil,
				Paginator: backPaginator1,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List second part of all workers (backward).
		{
			name:    prefix + "paginator-back2",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter:    nil,
				Sorter:    nil,
				Paginator: backPaginator2,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// No matches.
		{
			name:    prefix + "nomatch",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: missingFilter,
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// Error in filter.
		{
			name:    prefix + "bad-filter",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: badFilter,
				Sorter: nil,
			})),
			contentType: contentTypeJSON,
			status:      http.StatusBadRequest,
		},
		// Bad sort item.
		{
			name:    prefix + "bad-sorter-item",
			path:    filterPath,
			methods: methods,
			json: string(jsonMustMarshal(util.WorkersListSpec{
				Filter: validFilter,
				Sorter: sorterBad,
			})),
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

	tests := []requestTest{
		{
			name:        prefix + "valid",
			path:        path + validUUID,
			methods:     methods,
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
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

	tests := []requestTest{
		{
			name:        "api-version",
			path:        "/api/v1/version",
			methods:     []string{http.MethodGet, http.MethodGet},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
	}

	runTests(assert, r, tests)
}
