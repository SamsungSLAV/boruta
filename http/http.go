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

// Package http provides datatypes that are shared between server and client.
package http

import (
	"net"
	"net/http"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
)

const (
	// DateFormat denotes layout of timestamps used by Boruta HTTP API.
	DateFormat = time.RFC3339
)

// Headers returned by Boruta.
const (
	// ListTotalItemsHdr is HTTP header that is set as part of response to filtering methods.
	// It contains number of items that match a filter.
	ListTotalItemsHdr = "Boruta-List-Total"
	// ListRemainingItemsHdr is HTTP header that is set in response to filtering methods. It contains
	// number of items that weren't paged through yet.
	ListRemainingItemsHdr = "Boruta-List-Remaining"
	// RequestStateHdr is HTTP header that is set in responses that (would) carry boruta.ReqInfo.
	// It contains state of the request.
	RequestStateHdr = "Boruta-Request-State"
	// RequestCountHdr is HTTP header that is set in responses that (would) carry collection of
	// boruta.ReqInfo items. It contains number of items in the collection.
	RequestCountHdr = "Boruta-Request-Count"
	// JobTimeoutHdr is HTTP header that is set in responses that (would) carry boruta.ReqInfo
	// of request that is in INPROGRESS state. It denotes when timeout will pass and request will
	// be closed.
	JobTimeoutHdr = "Boruta-Job-Timeout"
	// WorkerStateHdr is HTTP header that is set in responses that (would) carry
	// boruta.WorkerInfo. It contains state of the worker.
	WorkerStateHdr = "Boruta-Worker-State"
	// WorkerCountHdr is HTTP header that is set in repsonses that (would) carry collection of
	// boruta.WorkerInfo items.
	WorkerCountHdr = "Boruta-Worker-Count"
	// ServerVersionHdr is HTTP header that is set when Boruta version information is requested.
	// It contains version of the server.
	ServerVersionHdr = "Boruta-Server-Version"
	// APIVersionHdr is HTTP header that is set when Boruta version information is requested. It
	// contains version of the HTTP API.
	APIVersionHdr = "Boruta-API-Version"
	// APIStateHdr is HTTP header that is set when Boruta version information is requested. It
	// contains state of used API version.
	APIStateHdr = "Boruta-API-State"
)

// API possible states.
const (
	// Devel means that API is in active development and changes may occur.
	Devel = "devel"
	// Stable means that there won't be any changes in the API.
	Stable = "stable"
	// Deprecated means that there is newer stable version of API and this version may be
	// removed in the future.
	Deprecated = "deprecated"
)

// RequestsListSpec is intended for (un)marshaling ListRequests parameters in HTTP API.
type RequestsListSpec struct {
	// Filter contains information how to filter list of requests.
	Filter *filter.Requests
	// Sorter contains SortInfo data.
	Sorter *boruta.SortInfo
	// Paginator contains RequestsPaginator.
	Paginator *boruta.RequestsPaginator
}

// WorkersListSpec is intended for (un)marshaling ListWorkers parameters in HTTP API.
type WorkersListSpec struct {
	// Filter contains information how to filter list of workers.
	Filter *filter.Workers
	// Sorter contains SortInfo data.
	Sorter *boruta.SortInfo
	// Paginator contains WorkersPaginator.
	Paginator *boruta.WorkersPaginator
}

// ReqIDPack is used for JSON (un)marshaller.
type ReqIDPack struct {
	boruta.ReqID
}

// WorkerStatePack is used by JSON (un)marshaller.
type WorkerStatePack struct {
	boruta.WorkerState
}

// AccessInfo2 structure is used by HTTP instead of AccessInfo when acquiring
// worker. The only difference is that key field is in PEM format instead of
// rsa.PrivateKey. It is temporary solution - session private keys will be
// replaces with users' public keys when proper user support is added.
type AccessInfo2 struct {
	// Addr is necessary information to connect to a tunnel to Dryad.
	Addr *net.TCPAddr
	// Key is private RSA key in PEM format.
	Key string
	// Username is a login name for the job session.
	Username string
}

// BorutaVersion contains information about server and API version.
type BorutaVersion struct {
	Server string
	API    string `json:"API_Version"`
	State  string `json:"API_State"`
}

// Response is desired to be used by HTTP API handlers as a return value.
type Response struct {
	// Data contains actual content (e.g. pointer to structure, error) that will be marshalled
	// and passed in HTTP response body.
	Data interface{}
	// Additional headers that will be added to HTTP response.
	Headers http.Header
}

// NewResponse takes data of any type and http.Headers and prepares pointer to Response structure,
// which should be returned by HTTP API handler function.
func NewResponse(data interface{}, headers http.Header) *Response {
	switch data := data.(type) {
	// Don't clasify ServerError as an error.
	case *ServerError:
	case ServerError:
	case error:
		return &Response{
			Data:    NewServerError(data),
			Headers: headers,
		}
	}
	return &Response{
		Data:    data,
		Headers: headers,
	}
}
