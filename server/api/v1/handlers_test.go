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
	"net/http"
	"testing"
)

func TestNewRequestHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "new-req",
			path:        "/api/v1/reqs/",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
	}

	runTests(assert, api, tests)
}

func TestCloseRequestHandler(t *testing.T) {
	assert, m, api := initTest(t)
	defer m.finish()

	tests := []requestTest{
		{
			name:        "close-req",
			path:        "/api/v1/reqs/8/close",
			methods:     []string{http.MethodPost},
			json:        ``,
			contentType: contentTypeJSON,
			status:      http.StatusNotImplemented,
		},
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
