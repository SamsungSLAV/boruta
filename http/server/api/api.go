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

// Package api aggregates all availabe Boruta HTTP API versions.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SamsungSLAV/boruta"
	util "github.com/SamsungSLAV/boruta/http"
	"github.com/SamsungSLAV/boruta/http/server/api/v1"
	"github.com/SamsungSLAV/slav/logger"
	"github.com/dimfeld/httptreemux"
)

// defaultAPI contains information which version of the API is treated as default.
// It should always be latest stable version.
const defaultAPI = v1.Version

// API provides HTTP API handlers.
type API struct {
	r       *httptreemux.TreeMux
	reqs    boruta.Requests
	workers boruta.Workers
}

// panicHandler is desired as httptreemux PanicHandler function. It sends
// InternalServerError with details to client whose request caused panic.
func panicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	var reason interface{}
	var status = http.StatusInternalServerError
	switch srvErr := err.(type) {
	case *util.ServerError:
		reason = srvErr.Err
		status = srvErr.Status
	default:
		reason = srvErr
	}
	// Because marshalling JSON may fail, data is sent in plaintext.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("Internal Server Error:\n%s", reason)))
	logger.WithError(err).Notice("httptreemux panic was handled")
}

// redirectToDefault redirects requests which lack API version information to
// default API. For example, if "v1" is the default API version, then request
// with path "/api/reqs/list" will be redirected to "/api/v1/reqs/list".
func redirectToDefault(w http.ResponseWriter, r *http.Request,
	p map[string]string) {
	u := *r.URL
	u.Path = "/api/" + defaultAPI + "/" + p["path"]
	http.Redirect(w, r, u.String(), http.StatusPermanentRedirect)
}

// setNotFoundHandler catches all requests that were redirected to default API,
// and not found there.
func notFoundHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	srvErr := util.NewServerError(boruta.NotFoundError(r.URL.Path))
	data, err := json.Marshal(srvErr)
	if err != nil {
		data = []byte(srvErr.Err)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(srvErr.Status)
	w.Write(data)
}

// setDefaultAPI register handler for API calls that lack API version in path.
func setDefaultAPIRedirect(prefix *httptreemux.Group) {
	for _, method := range [...]string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	} {
		prefix.Handle(method, "/*path", redirectToDefault)
		// Redirect was done, requested API call wasn't found.
		prefix.Handle(method, "/"+defaultAPI+"/*path", notFoundHandler)
	}
}

// NewAPI registers all available Boruta HTTP APIs on provided router. It also
// sets panicHandler for all panics that may occur in any API. Finally it sets
// default API version to which requests that miss API version are redirected.
func NewAPI(router *httptreemux.TreeMux, requestsAPI boruta.Requests,
	workersAPI boruta.Workers) (api *API) {
	api = new(API)

	api.reqs = requestsAPI
	api.workers = workersAPI

	api.r = router
	api.r.PanicHandler = panicHandler
	api.r.RedirectBehavior = httptreemux.Redirect308

	all := api.r.NewGroup("/api")
	v1group := all.NewGroup("/" + v1.Version)

	_ = v1.NewAPI(v1group, api.reqs, api.workers)
	setDefaultAPIRedirect(all)

	return
}
