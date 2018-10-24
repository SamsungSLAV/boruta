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

// File http/server/api/v1/handlers.go contain all handlers that are used in v1 API.

package v1

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	util "github.com/SamsungSLAV/boruta/http"
)

// newRequestHandler parses HTTP request for creating new Boruta request and
// calls NewRequest().
func (api *API) newRequestHandler(r *http.Request, ps map[string]string) responseData {
	var newReq boruta.ReqInfo
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&newReq); err != nil {
		return util.NewServerError(err)
	}

	//FIXME: currently UserInfo is ignored. Change when user support is added.
	rid, err := api.reqs.NewRequest(newReq.Caps, newReq.Priority, boruta.UserInfo{},
		newReq.ValidAfter.UTC(), newReq.Deadline.UTC())
	if err != nil {
		return util.NewServerError(err)
	}

	return util.ReqIDPack{ReqID: rid}
}

// closeRequestHandler parses HTTP request for closing existing Boruta request
// and calls CloseRequest().
func (api *API) closeRequestHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewServerError(util.ErrBadID)
	}

	return util.NewServerError(api.reqs.CloseRequest(reqid))
}

// updateRequestHandler parses HTTP request for modification of existing Boruta
// request and calls appropriate methods: SetRequestValidAfter(),
// SetRequestDeadline() and SetRequestPriority().
func (api *API) updateRequestHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewServerError(util.ErrBadID)
	}

	var req boruta.ReqInfo
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return util.NewServerError(err)
	}

	if req.ID != 0 && req.ID != reqid {
		return util.NewServerError(util.ErrIDMismatch)
	}
	// When ID wasn't set in JSON (or was set to 0) then it should be set to
	// the value from URL.
	req.ID = reqid

	return util.NewServerError(api.reqs.UpdateRequest(&req))
}

// getRequestInfoHandler parses HTTP request for getting information about Boruta
// request and calls GetRequestInfo().
func (api *API) getRequestInfoHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewServerError(util.ErrBadID)
	}

	reqinfo, err := api.reqs.GetRequestInfo(reqid)
	if err != nil {
		return util.NewServerError(err)
	}

	return reqinfo
}

// listRequestsHandler parses HTTP request for listing Boruta requests and calls
// ListRequests().
func (api *API) listRequestsHandler(r *http.Request, ps map[string]string) responseData {
	var rfilter *filter.Requests
	defer r.Body.Close()

	if r.Method == http.MethodPost {
		rfilter = new(filter.Requests)
		if err := json.NewDecoder(r.Body).Decode(rfilter); err != nil {
			if err != io.EOF {
				return util.NewServerError(err)
			}
			rfilter = nil
		}
	}

	reqs, err := api.reqs.ListRequests(rfilter)
	if err != nil {
		return util.NewServerError(err)
	}

	return reqs
}

// acquireWorkerHandler parses HTTP request for acquiring worker for Boruta
// request and calls AcquireWorker().
func (api *API) acquireWorkerHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewServerError(util.ErrBadID)
	}

	accessInfo, err := api.reqs.AcquireWorker(reqid)
	if err != nil {
		return util.NewServerError(err)
	}
	key := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(&accessInfo.Key),
	}))
	return util.AccessInfo2{
		Addr:     accessInfo.Addr.(*net.TCPAddr),
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
		return util.NewServerError(util.ErrBadID)
	}

	return util.NewServerError(api.reqs.ProlongAccess(reqid))
}

// listWorkersHandler parses HTTP request for listing workers and calls ListWorkers().
func (api *API) listWorkersHandler(r *http.Request, ps map[string]string) responseData {
	var wfilter filter.Workers
	defer r.Body.Close()

	if r.Method == http.MethodPost {
		err := json.NewDecoder(r.Body).Decode(&wfilter)
		if err != nil && err != io.EOF {
			return util.NewServerError(err)
		}
	}

	workers, err := api.workers.ListWorkers(wfilter.Groups, wfilter.Capabilities)
	if err != nil {
		return util.NewServerError(err)
	}

	return workers
}

// getWorkerInfoHandler parses HTTP request for obtaining worker information and
// calls GetWorkerInfo().
func (api *API) getWorkerInfoHandler(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewServerError(util.ErrBadUUID)
	}

	workerinfo, err := api.workers.GetWorkerInfo(boruta.WorkerUUID(ps["id"]))
	if err != nil {
		return util.NewServerError(err)
	}

	return workerinfo
}

// setWorkerStateHandler parses HTTP workers for setting worker state and calls
// workers.SetState().
func (api *API) setWorkerStateHandler(r *http.Request, ps map[string]string) responseData {
	var state util.WorkerStatePack
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewServerError(util.ErrBadUUID)
	}

	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		return util.NewServerError(err)
	}

	return util.NewServerError(api.workers.SetState(boruta.WorkerUUID(ps["id"]),
		state.WorkerState))
}

// setWorkerGroupsHandler parses HTTP workers for setting worker groups and calls
// workers.SetGroups().
func (api *API) setWorkerGroupsHandler(r *http.Request, ps map[string]string) responseData {
	var groups boruta.Groups
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewServerError(util.ErrBadUUID)
	}

	if err := json.NewDecoder(r.Body).Decode(&groups); err != nil {
		return util.NewServerError(err)
	}
	return util.NewServerError(api.workers.SetGroups(boruta.WorkerUUID(ps["id"]), groups))
}

// workerDeregister parses HTTP workers for deregistering worker state and calls
// workers.Deregister().
func (api *API) workerDeregister(r *http.Request, ps map[string]string) responseData {
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewServerError(util.ErrBadUUID)
	}

	return util.NewServerError(api.workers.Deregister(boruta.WorkerUUID(ps["id"])))
}

func (api *API) versionHandler(r *http.Request, ps map[string]string) responseData {
	return &util.BorutaVersion{
		Server: boruta.Version,
		API:    Version,
		State:  State,
	}
}
