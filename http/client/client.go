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

// Package client provides methods for interaction with Boruta REST API server.
//
// Provided BorutaClient type besides implementing boruta.Requests and
// boruta.Workers interfaces provides few convenient methods that allow to
// quickly check boruta.Request state, timeout and boruta.Worker state.
package client

import (
	"time"

	"git.tizen.org/tools/boruta"
	util "git.tizen.org/tools/boruta/http"
)

// BorutaClient handles interaction with specified Boruta server.
type BorutaClient struct {
	url string
	boruta.Requests
	boruta.Workers
}

// apiPrefix is part of URL that is common in all uses and contains API
// version.
const apiPrefix = "/api/v1/"

// NewBorutaClient provides BorutaClient ready to communicate with specified
// Boruta server.
//
//	cl := NewBorutaClient("http://127.0.0.1:1234")
func NewBorutaClient(url string) *BorutaClient {
	return &BorutaClient{
		url: url + apiPrefix,
	}
}

// NewRequest creates new Boruta request.
func (client *BorutaClient) NewRequest(caps boruta.Capabilities,
	priority boruta.Priority, owner boruta.UserInfo, validAfter time.Time,
	deadline time.Time) (boruta.ReqID, error) {

	return boruta.ReqID(0), util.ErrNotImplemented
}

// CloseRequest closes or cancels Boruta request.
func (client *BorutaClient) CloseRequest(reqID boruta.ReqID) error {
	return util.ErrNotImplemented
}

// UpdateRequest prepares JSON with fields that should be changed for given
// request ID.
func (client *BorutaClient) UpdateRequest(reqInfo *boruta.ReqInfo) error {
	return util.ErrNotImplemented
}

// GetRequestInfo queries Boruta server for details about given request ID.
func (client *BorutaClient) GetRequestInfo(reqID boruta.ReqID) (*boruta.ReqInfo,
	error) {
	return nil, util.ErrNotImplemented
}

// ListRequests queries Boruta server for list of requests that match given
// filter. Filter may be empty or nil to get list of all requests.
func (client *BorutaClient) ListRequests(filter boruta.ListFilter) (
	[]boruta.ReqInfo, error) {
	return nil, util.ErrNotImplemented
}

// AcquireWorker queries Boruta server for information required to access
// assigned Dryad. Access information may not be available when the call
// is issued because requests need to have assigned worker.
func (client *BorutaClient) AcquireWorker(reqID boruta.ReqID) (
	*boruta.AccessInfo, error) {
	return nil, util.ErrNotImplemented
}

// ProlongAccess requests Boruta server to extend running time of job. User may
// need to call this method multiple times as long as access to Dryad is needed.
// If not called, Boruta server will terminate the tunnel when ReqInfo.Job.Timeout
// passes, and change state of request to CLOSED.
func (client *BorutaClient) ProlongAccess(reqID boruta.ReqID) error {
	return util.ErrNotImplemented
}

// ListWorkers queries Boruta server for list of workers that are in given groups
// and have provided capabilities. Setting both caps and groups to empty or nil
// lists all workers.
func (client *BorutaClient) ListWorkers(groups boruta.Groups,
	caps boruta.Capabilities) ([]boruta.WorkerInfo, error) {
	return nil, util.ErrNotImplemented
}

// GetWorkerInfo queries Boruta server for information about worker with given
// UUID.
func (client *BorutaClient) GetWorkerInfo(uuid boruta.WorkerUUID) (
	boruta.WorkerInfo, error) {
	return boruta.WorkerInfo{}, util.ErrNotImplemented
}

// SetState requests Boruta server to change state of worker with provided UUID.
// SetState is intended only for Boruta server administrators.
func (client *BorutaClient) SetState(uuid boruta.WorkerUUID,
	state boruta.WorkerState) error {
	return util.ErrNotImplemented
}

// SetGroups requests Boruta server to change groups of worker with provided
// UUID. SetGroups is intended only for Boruta server administrators.
func (client *BorutaClient) SetGroups(uuid boruta.WorkerUUID,
	groups boruta.Groups) error {
	return util.ErrNotImplemented
}

// Deregister requests Boruta server to deregister worker with provided UUID.
// Deregister is intended only for Boruta server administrators.
func (client *BorutaClient) Deregister(uuid boruta.WorkerUUID) error {
	return util.ErrNotImplemented
}
