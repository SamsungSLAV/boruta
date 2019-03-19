/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
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

// File http/http_test.go provides test of Boruta HTTP server errors.

package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

func TestNewResponse(t *testing.T) {
	assert := assert.New(t)

	resp := NewResponse(nil, nil)
	assert.Nil(resp.Data)
	assert.Nil(resp.Headers)

	rid := boruta.ReqID(1)
	resp = NewResponse(rid, nil)
	assert.Equal(rid, resp.Data)
	assert.Nil(resp.Headers)

	headers := make(http.Header)
	headers.Set("foo", "bar")
	resp = NewResponse(nil, headers)
	assert.Nil(resp.Data)
	assert.Equal(headers, resp.Headers)

	resp = NewResponse(rid, headers)
	assert.Equal(rid, resp.Data)
	assert.Equal(headers, resp.Headers)

	err := errors.New("foo")
	srvErrPtr := NewServerError(err)

	resp = NewResponse(err, nil)
	assert.Equal(srvErrPtr, resp.Data)
	assert.Nil(resp.Headers)

	resp = NewResponse(err, headers)
	assert.Equal(srvErrPtr, resp.Data)
	assert.Equal(headers, resp.Headers)

	resp = NewResponse(srvErrPtr, nil)
	assert.Equal(srvErrPtr, resp.Data)
	assert.Nil(resp.Headers)

	resp = NewResponse(srvErrPtr, headers)
	assert.Equal(srvErrPtr, resp.Data)
	assert.Equal(headers, resp.Headers)

	srvErr := ServerError{Err: "foo", Status: http.StatusBadRequest}
	resp = NewResponse(srvErr, nil)
	assert.Equal(srvErr, resp.Data)
	assert.Nil(resp.Headers)

	resp = NewResponse(srvErr, headers)
	assert.Equal(srvErr, resp.Data)
	assert.Equal(headers, resp.Headers)
}
