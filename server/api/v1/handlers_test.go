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

	methods := []string{http.MethodGet, http.MethodHead}
	prefix := "req-info-"
	path := "/api/v1/reqs/"

	var req ReqInfo
	err := json.Unmarshal([]byte(validReqJSON), &req)
	assert.Nil(err)

	notFoundTest := testFromTempl(notFoundTestTempl, prefix, path+"2", methods...)
	invalidIDTest := testFromTempl(invalidIDTestTempl, prefix, path+invalidID, methods...)
	m.rq.EXPECT().GetRequestInfo(ReqID(1)).Return(req, nil).Times(2)
	m.rq.EXPECT().GetRequestInfo(ReqID(2)).Return(ReqInfo{}, NotFoundError("Request")).Times(2)

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
		// Try to get request information of request that doesn't exist.
		notFoundTest,
		// Try to get request with invalid ID.
		invalidIDTest,
	}

	runTests(assert, api, tests)
}

func TestListRequestsHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	deadline, err := time.Parse(dateLayout, future)
	assert.Nil(err)
	validAfter, err := time.Parse(dateLayout, past)
	assert.Nil(err)
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

	methods := []string{http.MethodPost}
	prefix := "filter-reqs-"
	filterPath := "/api/v1/reqs/list"
	malformedJSONTest := testFromTempl(malformedJSONTestTempl, prefix, filterPath, methods...)

	validFilter := NewRequestFilter("WAIT", "")
	m.rq.EXPECT().ListRequests(validFilter).Return(reqs[:2], nil)

	emptyFilter := NewRequestFilter("", "")
	m.rq.EXPECT().ListRequests(emptyFilter).Return(reqs, nil).Times(2)
	m.rq.EXPECT().ListRequests(nil).Return(reqs, nil).Times(3)

	missingFilter := NewRequestFilter("INVALID", "")
	m.rq.EXPECT().ListRequests(missingFilter).Return([]ReqInfo{}, nil)

	// Currently ListRequests doesn't return any error hence the meaningless values.
	badFilter := NewRequestFilter("FAIL", "-1")
	m.rq.EXPECT().ListRequests(badFilter).Return([]ReqInfo{}, errors.New("foo bar: pizza failed"))

	tests := []requestTest{
		// Valid filter - list some requests.
		{
			name:        prefix + "valid-filter",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(validFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// List all requests.
		{
			name:        "list-reqs-all",
			path:        "/api/v1/reqs/",
			methods:     []string{http.MethodGet, http.MethodHead},
			json:        ``,
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
			name:        prefix + "empty",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(emptyFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
		},
		// No matches
		{
			name:        prefix + "nomatch-all",
			path:        filterPath,
			methods:     methods,
			json:        string(jsonMustMarshal(missingFilter)),
			contentType: contentTypeJSON,
			status:      http.StatusOK,
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

	runTests(assert, api, tests)
}

func TestAcquireWorkerHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	prefix := "acquire-worker-"
	pathfmt := "/api/v1/reqs/%s/acquire_worker"

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

	m.rq.EXPECT().AcquireWorker(ReqID(1)).Return(access, nil)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().AcquireWorker(ReqID(2)).Return(AccessInfo{}, NotFoundError("Request"))
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

	runTests(assert, api, tests)
}

func TestProlongAccessHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	methods := []string{http.MethodPost}
	prefix := "prolong-access-"
	pathfmt := "/api/v1/reqs/%s/prolong"

	m.rq.EXPECT().ProlongAccess(ReqID(1)).Return(nil)
	notFoundTest := testFromTempl(notFoundTestTempl, prefix, fmt.Sprintf(pathfmt, "2"), methods...)
	m.rq.EXPECT().ProlongAccess(ReqID(2)).Return(NotFoundError("Request"))
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
