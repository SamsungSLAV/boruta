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

// Package v1 provides HTTP API version 1 of Boruta. Through this API clients may:
// * list, create, manage and get details of requests;
// * list, acquire, prolong access to and get details of workers.
package v1

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/SamsungSLAV/boruta"
	util "github.com/SamsungSLAV/boruta/http"
	"github.com/dimfeld/httptreemux"
)

// reqHandler denotes function that parses HTTP request and returns pointer to util.Response.
type reqHandler func(*http.Request, map[string]string) *util.Response

// Version contains version string of the API.
const Version = "v1"

// State contains information about state of the API (devel, stable or obsolete).
const State = util.Devel

// API provides HTTP API handlers.
type API struct {
	r       *httptreemux.Group
	reqs    boruta.Requests
	workers boruta.Workers
}

// uuidRE matches only valid UUID strings.
var uuidRE = regexp.MustCompile("^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$")

// jsonMustMarshal tries to marshal passed data to JSON. Panics if error occurs.
func jsonMustMarshal(data interface{}) []byte {
	res, err := json.Marshal(data)
	if err != nil {
		msg := "unable to marshal JSON:" + err.Error()
		panic(util.NewServerError(util.ErrInternalServerError, msg))
	}
	return res
}

// routerSetHandler wraps fn by adding HTTP headers, handling error and
// marshalling. Such wrapped function is then registered in the API router as a
// handler for given path, provided methods and HTTP success status that should
// be used when funcion succeeds.
func routerSetHandler(grp *httptreemux.Group, path string, fn reqHandler,
	status int, methods ...string) {
	newHandler := func(handle reqHandler) httptreemux.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request,
			ps map[string]string) {
			status := status
			respoonse := handle(r, ps)
			switch data := respoonse.Data.(type) {
			case *util.ServerError:
				if data != nil {
					status = data.Status
				}
			case boruta.ReqInfo:
				w.Header().Add(util.RequestStateHdr, string(data.State))
				if data.State == boruta.INPROGRESS {
					w.Header().Add(util.JobTimeoutHdr,
						data.Job.Timeout.Format(util.DateFormat))
				}
			case []boruta.ReqInfo:
				w.Header().Add(util.RequestCountHdr, strconv.Itoa(len(data)))
			case boruta.WorkerInfo:
				w.Header().Add(util.WorkerStateHdr, string(data.State))
			case []boruta.WorkerInfo:
				w.Header().Add(util.WorkerCountHdr, strconv.Itoa(len(data)))
			case *util.BorutaVersion:
				w.Header().Add(util.ServerVersionHdr, data.Server)
				w.Header().Add(util.APIVersionHdr, data.API)
				w.Header().Add(util.APIStateHdr, data.State)
			}
			if status != http.StatusNoContent {
				w.Header().Set("Content-Type", "application/json")
			}
			for k, v := range respoonse.Headers {
				for _, s := range v {
					w.Header().Add(k, s)
				}
			}
			w.WriteHeader(status)
			if status != http.StatusNoContent {
				w.Write(jsonMustMarshal(respoonse.Data))
			}
		}
	}
	for _, method := range methods {
		grp.Handle(method, path, newHandler(fn))
	}
}

// NewAPI takes router and registers HTTP API in it. htttreemux.PanicHandler
// function is set. Also other setting of the router may be modified.
func NewAPI(router *httptreemux.Group, requestsAPI boruta.Requests,
	workersAPI boruta.Workers) (api *API) {
	api = new(API)

	api.reqs = requestsAPI
	api.workers = workersAPI

	api.r = router

	main := api.r.NewGroup("/")
	reqs := api.r.NewGroup("/reqs")
	workers := api.r.NewGroup("/workers")

	// Requests API
	routerSetHandler(reqs, "/list", api.listRequestsHandler, http.StatusOK,
		http.MethodPost)
	routerSetHandler(reqs, "/", api.newRequestHandler, http.StatusCreated,
		http.MethodPost)
	routerSetHandler(reqs, "/:id", api.getRequestInfoHandler, http.StatusOK,
		http.MethodGet, http.MethodHead)
	routerSetHandler(reqs, "/:id", api.updateRequestHandler,
		http.StatusNoContent, http.MethodPost)
	routerSetHandler(reqs, "/:id/close", api.closeRequestHandler,
		http.StatusNoContent, http.MethodPost)
	routerSetHandler(reqs, "/:id/acquire_worker", api.acquireWorkerHandler,
		http.StatusOK, http.MethodPost)
	routerSetHandler(reqs, "/:id/prolong", api.prolongAccessHandler,
		http.StatusNoContent, http.MethodPost)

	// Workers API
	routerSetHandler(workers, "/", api.listWorkersHandler, http.StatusOK,
		http.MethodGet, http.MethodHead)
	routerSetHandler(workers, "/list", api.listWorkersHandler, http.StatusOK,
		http.MethodPost)
	routerSetHandler(workers, "/:id", api.getWorkerInfoHandler, http.StatusOK,
		http.MethodGet, http.MethodHead)

	// Workers API - Admin part
	routerSetHandler(workers, "/:id/setstate", api.setWorkerStateHandler,
		http.StatusAccepted, http.MethodPost)
	routerSetHandler(workers, "/:id/setgroups", api.setWorkerGroupsHandler,
		http.StatusNoContent, http.MethodPost)
	routerSetHandler(workers, "/:id/deregister", api.workerDeregister,
		http.StatusNoContent, http.MethodPost)

	// other functions
	routerSetHandler(main, "/version", api.versionHandler, http.StatusOK, http.MethodGet,
		http.MethodHead)

	return
}

func parseReqID(id string) (boruta.ReqID, error) {
	reqid, err := strconv.ParseUint(id, 10, 64)
	return boruta.ReqID(reqid), err
}

// isValidUUID checks if given string is properly formatted UUID.
func isValidUUID(id string) bool {
	return uuidRE.MatchString(id)
}
