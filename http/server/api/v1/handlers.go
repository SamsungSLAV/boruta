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
func (api *API) newRequestHandler(r *http.Request, ps map[string]string) *util.Response {
	var newReq boruta.ReqInfo
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&newReq); err != nil {
		return util.NewResponse(err, nil)
	}

	//FIXME: currently UserInfo is ignored. Change when user support is added.
	rid, err := api.reqs.NewRequest(newReq.Caps, newReq.Priority, boruta.UserInfo{},
		newReq.ValidAfter.UTC(), newReq.Deadline.UTC())
	if err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(util.ReqIDPack{ReqID: rid}, nil)
}

// closeRequestHandler parses HTTP request for closing existing Boruta request
// and calls CloseRequest().
func (api *API) closeRequestHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewResponse(util.ErrBadID, nil)
	}

	return util.NewResponse(api.reqs.CloseRequest(reqid), nil)
}

// updateRequestHandler parses HTTP request for modification of existing Boruta
// request and calls appropriate methods: SetRequestValidAfter(),
// SetRequestDeadline() and SetRequestPriority().
func (api *API) updateRequestHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewResponse(util.ErrBadID, nil)
	}

	var req boruta.ReqInfo
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return util.NewResponse(err, nil)
	}

	if req.ID != 0 && req.ID != reqid {
		return util.NewResponse(util.ErrIDMismatch, nil)
	}
	// When ID wasn't set in JSON (or was set to 0) then it should be set to
	// the value from URL.
	req.ID = reqid

	return util.NewResponse(api.reqs.UpdateRequest(&req), nil)
}

// getRequestInfoHandler parses HTTP request for getting information about Boruta
// request and calls GetRequestInfo().
func (api *API) getRequestInfoHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewResponse(util.ErrBadID, nil)
	}

	reqinfo, err := api.reqs.GetRequestInfo(reqid)
	if err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(reqinfo, nil)
}

// listRequestsHandler parses HTTP request for listing Boruta requests and calls
// ListRequests().
func (api *API) listRequestsHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	listSpec := &util.RequestsListSpec{}

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(listSpec); err != nil {
			if err != io.EOF {
				return util.NewResponse(err, nil)
			}
			listSpec.Filter = nil
			listSpec.Sorter = nil
		}
	}

	reqs, err := api.reqs.ListRequests(listSpec.Filter, listSpec.Sorter)
	if err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(reqs, nil)
}

// acquireWorkerHandler parses HTTP request for acquiring worker for Boruta
// request and calls AcquireWorker().
func (api *API) acquireWorkerHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewResponse(util.ErrBadID, nil)
	}

	accessInfo, err := api.reqs.AcquireWorker(reqid)
	if err != nil {
		return util.NewResponse(err, nil)
	}
	key := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(&accessInfo.Key),
	}))
	return util.NewResponse(util.AccessInfo2{
		Addr:     accessInfo.Addr.(*net.TCPAddr),
		Key:      key,
		Username: accessInfo.Username,
	}, nil)
}

// prolongAccessHandler parses HTTP request for prolonging previously acquired
// worker and calls ProlongAccess().
func (api *API) prolongAccessHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	reqid, err := parseReqID(ps["id"])
	if err != nil {
		return util.NewResponse(util.ErrBadID, nil)
	}

	return util.NewResponse(api.reqs.ProlongAccess(reqid), nil)
}

// listWorkersHandler parses HTTP request for listing workers and calls ListWorkers().
func (api *API) listWorkersHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	listSpec := &util.WorkersListSpec{}
	// Read list spec.
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(listSpec); err != nil {
			if err != io.EOF {
				return util.NewResponse(err, nil)
			}
			listSpec.Filter = nil
			listSpec.Sorter = nil
		}
	}

	// WorkersFilter needs to be regenerated to initialize internal structures.
	if listSpec.Filter != nil {
		listSpec.Filter = filter.NewWorkers(listSpec.Filter.Groups,
			listSpec.Filter.Capabilities)
	}
	// Filter and sort workers.
	workers, err := api.workers.ListWorkers(listSpec.Filter, listSpec.Sorter)
	if err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(workers, nil)
}

// getWorkerInfoHandler parses HTTP request for obtaining worker information and
// calls GetWorkerInfo().
func (api *API) getWorkerInfoHandler(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewResponse(util.ErrBadUUID, nil)
	}

	workerinfo, err := api.workers.GetWorkerInfo(boruta.WorkerUUID(ps["id"]))
	if err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(workerinfo, nil)
}

// setWorkerStateHandler parses HTTP workers for setting worker state and calls
// workers.SetState().
func (api *API) setWorkerStateHandler(r *http.Request, ps map[string]string) *util.Response {
	var state util.WorkerStatePack
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewResponse(util.ErrBadUUID, nil)
	}

	if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
		return util.NewResponse(err, nil)
	}

	return util.NewResponse(api.workers.SetState(boruta.WorkerUUID(ps["id"]),
		state.WorkerState), nil)
}

// setWorkerGroupsHandler parses HTTP workers for setting worker groups and calls
// workers.SetGroups().
func (api *API) setWorkerGroupsHandler(r *http.Request, ps map[string]string) *util.Response {
	var groups boruta.Groups
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewResponse(util.ErrBadUUID, nil)
	}

	if err := json.NewDecoder(r.Body).Decode(&groups); err != nil {
		return util.NewResponse(err, nil)
	}
	return util.NewResponse(api.workers.SetGroups(boruta.WorkerUUID(ps["id"]), groups), nil)
}

// workerDeregister parses HTTP workers for deregistering worker state and calls
// workers.Deregister().
func (api *API) workerDeregister(r *http.Request, ps map[string]string) *util.Response {
	defer r.Body.Close()

	if !isValidUUID(ps["id"]) {
		return util.NewResponse(util.ErrBadUUID, nil)
	}

	return util.NewResponse(api.workers.Deregister(boruta.WorkerUUID(ps["id"])), nil)
}

func (api *API) versionHandler(r *http.Request, ps map[string]string) *util.Response {
	return util.NewResponse(&util.BorutaVersion{
		Server: boruta.Version,
		API:    Version,
		State:  State,
	}, nil)
}
