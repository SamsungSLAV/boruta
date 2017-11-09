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

// File http/errors.go provides errors that may occur when interacting with
// Boruta HTTP API.

package http

import (
	"errors"
	"io"
	"net/http"

	. "git.tizen.org/tools/boruta"
)

// ServerError represents error that occured while creating response.
type ServerError struct {
	// Err contains general error string.
	Err string `json:"error"`
	// Status contains HTTP error code that should be returned with the error.
	Status int `json:"-"`
}

var (
	// ErrNotImplemented is returned when requested functionality isn't
	// implemented yet.
	ErrNotImplemented = errors.New("not implemented yet")
	// ErrInternalServerError is returned when serious error in the server
	// occurs which isn't users' fault.
	ErrInternalServerError = errors.New("internal server error")
	// ErrBadRequest is returned when User request is invalid.
	ErrBadRequest = errors.New("invalid request")
	// ErrBadID is returned when User provided ID which can't be parsed into
	// uint.
	ErrBadID = errors.New("ID provided in URL isn't valid")
	// ErrBadUUID is returned when User provided ID which isn't valid UUID.
	ErrBadUUID = errors.New("ID provided in URL isn't valid UUID")
	// ErrIDMismatch is returned when User provided different ID values in
	// URL and JSON.
	ErrIDMismatch = errors.New("request ID set in JSON doesn't match ID from URL")
)

// isNotFoundError returns true if passed error is of type NotFoundError.
func isNotFoundError(err error) bool {
	_, ok := err.(NotFoundError)
	return ok
}

// NewServerError provides pointer to initialized ServerError.
func NewServerError(err error, details ...string) (ret *ServerError) {
	if err == nil {
		return nil
	}

	ret = new(ServerError)

	ret.Err = err.Error()
	if len(details) > 0 {
		ret.Err += ": " + details[0]
	}
	if isNotFoundError(err) {
		ret.Status = http.StatusNotFound
		return
	}

	switch err {
	case ErrNotImplemented:
		ret.Status = http.StatusNotImplemented
	case ErrInternalServerError:
		ret.Status = http.StatusInternalServerError
	case ErrBadRequest:
		ret.Status = http.StatusBadRequest
	case io.EOF:
		ret.Err = "no body provided in HTTP request"
		ret.Status = http.StatusBadRequest
	default:
		ret.Err = ErrBadRequest.Error() + ": " + ret.Err
		ret.Status = http.StatusBadRequest
	}

	return
}
