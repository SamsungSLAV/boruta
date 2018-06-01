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

	. "git.tizen.org/tools/boruta"
	util "git.tizen.org/tools/boruta/http"
	"github.com/dimfeld/httptreemux"
)

// responseData type denotes data returned by HTTP request handler functions.
// Returned values are directly converted to JSON responses.
type responseData interface{}

// reqHandler denotes function that parses HTTP request and returns responseData.
type reqHandler func(*http.Request, map[string]string) responseData

// Version contains version string of the API.
const Version = "v1"

// API provides HTTP API handlers.
type API struct {
	r       *httptreemux.Group
	reqs    Requests
	workers Workers
}

// uuidRE matches only valid UUID strings.
var uuidRE = regexp.MustCompile("^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$")

// jsonMustMarshal tries to marshal responseData to JSON. Panics if error occurs.
// TODO(mwereski): check type of data.
func jsonMustMarshal(data responseData) []byte {
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
			data := handle(r, ps)
			switch data := data.(type) {
			case *util.ServerError:
				if data != nil {
					status = data.Status
				}
			case ReqInfo:
				w.Header().Add("Boruta-Request-State", string(data.State))
				if data.State == INPROGRESS {
					w.Header().Add("Boruta-Job-Timeout",
						data.Job.Timeout.Format(util.DateFormat))
				}
			case []ReqInfo:
				w.Header().Add("Boruta-Request-Count", strconv.Itoa(len(data)))
			case WorkerInfo:
				w.Header().Add("Boruta-Worker-State", string(data.State))
			case []WorkerInfo:
				w.Header().Add("Boruta-Worker-Count", strconv.Itoa(len(data)))
			}
			if status != http.StatusNoContent {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(status)
			if status != http.StatusNoContent {
				w.Write(jsonMustMarshal(data))
			}
		}
	}
	for _, method := range methods {
		grp.Handle(method, path, newHandler(fn))
	}
}

// NewAPI takes router and registers HTTP API in it. htttreemux.PanicHandler
// function is set. Also other setting of the router may be modified.
func NewAPI(router *httptreemux.Group, requestsAPI Requests,
	workersAPI Workers) (api *API) {
	api = new(API)

	api.reqs = requestsAPI
	api.workers = workersAPI

	api.r = router

	reqs := api.r.NewGroup("/reqs")
	workers := api.r.NewGroup("/workers")

	// Requests API
	routerSetHandler(reqs, "/", api.listRequestsHandler, http.StatusOK,
		http.MethodGet, http.MethodHead)
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
		http.StatusNoContent, http.MethodPost)
	routerSetHandler(workers, "/:id/setgroups", api.setWorkerGroupsHandler,
		http.StatusNoContent, http.MethodPost)
	routerSetHandler(workers, "/:id/deregister", api.workerDeregister,
		http.StatusNoContent, http.MethodPost)

	return
}

func parseReqID(id string) (ReqID, error) {
	reqid, err := strconv.ParseUint(id, 10, 64)
	return ReqID(reqid), err
}

// isValidUUID checks if given string is properly formatted UUID.
func isValidUUID(id string) bool {
	return uuidRE.MatchString(id)
}
