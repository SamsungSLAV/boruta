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
	"fmt"
	"net/http"

	. "git.tizen.org/tools/boruta"
	"github.com/dimfeld/httptreemux"
)

// responseData type denotes data returned by HTTP request handler functions.
// Returned values are directly converted to JSON responses.
type responseData interface{}

// reqIDPack is used as input for JSON (un)marshaller.
type reqIDPack struct {
	ReqID
}

// reqHandler denotes function that parses HTTP request and returns responseData.
type reqHandler func(*http.Request, map[string]string) responseData

// API provides HTTP API handlers.
type API struct {
	r    *httptreemux.TreeMux
	reqs Requests
}

// jsonMustMarshal tries to marshal responseData to JSON. Panics if error occurs.
// TODO(mwereski): check type of data.
func jsonMustMarshal(data responseData) []byte {
	res, err := json.Marshal(data)
	if err != nil {
		msg := "unable to marshal JSON:" + err.Error()
		panic(newServerError(ErrInternalServerError, msg))
	}
	return res
}

// panicHandler is intended as a httptreemux PanicHandler function. It sends
// InternalServerError with details to client whose request caused panic.
func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	var reason interface{}
	var status = http.StatusInternalServerError
	switch srvErr := err.(type) {
	case *serverError:
		reason = srvErr.Err
		status = srvErr.Status
	default:
		reason = srvErr
	}
	// Because marshalling JSON may fail, data is sent in plaintext.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("Internal Server Error:\n%s", reason)))
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
			rdata := handle(r, ps)
			if data, isErr := rdata.(*serverError); isErr &&
				data != nil {
				status = data.Status
			}
			if status != http.StatusNoContent {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(status)
			if status != http.StatusNoContent {
				w.Write(jsonMustMarshal(rdata))
			}
		}
	}
	for _, method := range methods {
		grp.Handle(method, path, newHandler(fn))
	}
}

// NewAPI takes router and registers HTTP API in it. httptreemux.PanicHandler
// function is set. Also other setting of the router may be modified.
func NewAPI(router *httptreemux.TreeMux, requestsAPI Requests) (api *API) {
	api = new(API)

	api.reqs = requestsAPI

	api.r = router
	api.r.PanicHandler = panicHandler

	root := api.r.NewGroup("/api/v1")
	reqs := root.NewGroup("/reqs")
	workers := root.NewGroup("/workers")

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

	return
}
