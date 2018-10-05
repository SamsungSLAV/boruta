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

// File http/error_test.go provides test of Boruta HTTP server errors.

package http

import (
	"errors"
	"io"
	"net/http"
	"testing"

	. "github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

func TestNewServerError(t *testing.T) {
	assert := assert.New(t)
	badRequest := &ServerError{
		Err:    "invalid request: foo",
		Status: http.StatusBadRequest,
	}
	nobody := "no body provided in HTTP request"
	missingBody := &ServerError{
		Err:    nobody,
		Status: http.StatusBadRequest,
	}
	notImplemented := &ServerError{
		Err:    ErrNotImplemented.Error(),
		Status: http.StatusNotImplemented,
	}
	internalErr := &ServerError{
		Err:    ErrInternalServerError.Error(),
		Status: http.StatusInternalServerError,
	}
	customErr := &ServerError{
		Err:    "invalid request: more details",
		Status: http.StatusBadRequest,
	}
	notFound := &ServerError{
		Err:    NotFoundError("Fern Flower").Error(),
		Status: http.StatusNotFound,
	}
	assert.Equal(badRequest, NewServerError(errors.New("foo")))
	assert.Equal(missingBody, NewServerError(io.EOF))
	assert.Equal(notImplemented, NewServerError(ErrNotImplemented))
	assert.Equal(internalErr, NewServerError(ErrInternalServerError))
	assert.Equal(customErr, NewServerError(ErrBadRequest, "more details"))
	assert.Equal(notFound, NewServerError(NotFoundError("Fern Flower")))
	assert.Nil(NewServerError(nil))
}

func TestError(t *testing.T) {
	assert := assert.New(t)
	err := NewServerError(errors.New("foo"))
	assert.Equal(err.Err, err.Error())
}
