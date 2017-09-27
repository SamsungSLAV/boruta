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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/requests"
)

func TestNewRequestHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	prefix := "new-req-"
	path := "/api/v1/reqs/"
	methods := []string{http.MethodPost}
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, path, methods...)

	var req ReqInfo
	err := json.Unmarshal([]byte(validReqJSON), &req)
	assert.Nil(err)

	m.rq.EXPECT().NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter,
		req.Deadline).Return(ReqID(1), nil)
	m.rq.EXPECT().NewRequest(req.Caps, Priority(32), req.Owner, req.ValidAfter,
		req.Deadline).Return(ReqID(0), requests.ErrPriority)

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

	runTests(assert, api, tests)
}

func TestCloseRequestHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	pathfmt := "/api/v1/reqs/%s/close"
	prefix := "close-req-"

	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, fmt.Sprintf(pathfmt, invalidID), methods...)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().CloseRequest(ReqID(1)).Return(nil)
	m.rq.EXPECT().CloseRequest(ReqID(2)).Return(NotFoundError("Request"))

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

	runTests(assert, api, tests)
}

func TestUpdateRequestHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "update-req",
			path:        "/api/v1/reqs/8",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestGetRequestInfoHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "req-info",
			path:        "/api/v1/reqs/8",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestListRequestsHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "list-reqs-all",
			path:        "/api/v1/reqs/",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
		{
			name:        "list-reqs-filter",
			path:        "/api/v1/reqs/list",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestAcquireWorkerHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "acquire-worker",
			path:        "/api/v1/reqs/8/acquire_worker",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestProlongAccessHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "prolong-access",
			path:        "/api/v1/reqs/8/prolong",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestListWorkersHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "list-workers-all",
			path:        "/api/v1/workers/",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
		{
			name:        "list-workers-filter",
			path:        "/api/v1/workers/list",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestGetWorkerInfoHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "worker-info",
			path:        "/api/v1/workers/8",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}
