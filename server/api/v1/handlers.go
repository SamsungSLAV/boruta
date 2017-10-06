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

// File server/api/v1/handlers.go contains all handlers that are used in v1 API.

package v1

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"

	. "git.tizen.org/tools/boruta"
)

// newRequestHandler parses HTTP request for creating new Boruta request and
// calls NewRequest().
func (api *API) newRequestHandler(r *http.Request, ps map[string]string) responseData {
	var newReq ReqInfo
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&newReq); err != nil {
		return newServerError(err)
	}

	//FIXME: currently UserInfo is ignored. Change when user support is added.
	rid, err := api.reqs.NewRequest(newReq.Caps, newReq.Priority, UserInfo{},
		newReq.ValidAfter.UTC(), newReq.Deadline.UTC())
	if err != nil {
		return newServerError(err)
	}

	return reqIDPack{rid}
}

// closeRequestHandler parses HTTP request for closing existing Boruta request
// and calls CloseRequest().
func (api *API) closeRequestHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return newServerError(ErrBadID)
	}

	return newServerError(api.reqs.CloseRequest(reqid))
}

// updateRequestHandler parses HTTP request for modification of existing Boruta
// request and calls appropriate methods: SetRequestValidAfter(),
// SetRequestDeadline() and SetRequestPriority().
func (api *API) updateRequestHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return newServerError(ErrBadID)
	}

	var req ReqInfo
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return newServerError(err)
	}

	if req.ID != 0 && req.ID != reqid {
		return newServerError(ErrIDMismatch)
	}
	// When ID wasn't set in JSON (or was set to 0) then it should be set to
	// the value from URL.
	req.ID = reqid

	return newServerError(api.reqs.UpdateRequest(&req))
}

// getRequestInfoHandler parses HTTP request for getting information about Boruta
// request and calls GetRequestInfo().
func (api *API) getRequestInfoHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return newServerError(ErrBadID)
	}

	reqinfo, err := api.reqs.GetRequestInfo(reqid)
	if err != nil {
		return newServerError(err)
	}

	return reqinfo
}

// listRequestsHandler parses HTTP request for listing Boruta requests and calls
// ListRequests().
func (api *API) listRequestsHandler(r *http.Request, ps map[string]string) responseData {
	var filter *RequestFilter
	defer r.Body.Close()

	if r.Method == http.MethodPost {
		filter = new(RequestFilter)
		if err := json.NewDecoder(r.Body).Decode(filter); err != nil {
			if err != io.EOF {
				return newServerError(err)
			}
			filter = nil
		}
	}

	reqs, err := api.reqs.ListRequests(filter)
	if err != nil {
		return newServerError(err)
	}

	return reqs
}

// acquireWorkerHandler parses HTTP request for acquiring worker for Boruta
// request and calls AcquireWorker().
func (api *API) acquireWorkerHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return newServerError(ErrBadID)
	}

	accessInfo, err := api.reqs.AcquireWorker(reqid)
	if err != nil {
		return newServerError(err)
	}
	key := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(&accessInfo.Key),
	}))
	return AccessInfo2{
		Addr:     accessInfo.Addr,
		Key:      key,
		Username: accessInfo.Username,
	}
}

// prolongAccessHandler parses HTTP request for prolonging previously acquired
// worker and calls ProlongAccess().
func (api *API) prolongAccessHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return newServerError(ErrBadID)
	}

	return newServerError(api.reqs.ProlongAccess(reqid))
}

// listWorkersHandler parses HTTP request for listing workers and calls ListWorkers().
func (api *API) listWorkersHandler(r *http.Request, ps map[string]string) responseData {
	return newServerError(ErrNotImplemented, "list workers")
}

// getWorkerInfoHandler parses HTTP request for obtaining worker information and
// calls GetWorkerInfo().
func (api *API) getWorkerInfoHandler(r *http.Request, ps map[string]string) responseData {
	return newServerError(ErrNotImplemented, "get worker info")
}
