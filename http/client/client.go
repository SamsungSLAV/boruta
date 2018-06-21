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
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
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

const (
	// contentType denotes format in which we talk with Boruta server.
	contentType = "application/json"
	// apiPrefix is part of URL that is common in all uses and contains API
	// version.
	apiPrefix = "/api/v1/"
)

// NewBorutaClient provides BorutaClient ready to communicate with specified
// Boruta server.
//
//	cl := NewBorutaClient("http://127.0.0.1:1234")
func NewBorutaClient(url string) *BorutaClient {
	return &BorutaClient{
		url: url + apiPrefix,
	}
}

// readBody is simple wrapper function that reads body of http request into byte
// slice and closes the body.
func readBody(body io.ReadCloser) ([]byte, error) {
	defer body.Close()
	content, err := ioutil.ReadAll(body)
	if err != nil {
		err = errors.New("unable to read server response: " + err.Error())
	}
	return content, err
}

// bodyJSONUnmarshal is a wrapper that unmarshals server response into an
// appropriate structure.
func bodyJSONUnmarshal(body io.ReadCloser, val interface{}) error {
	content, err := readBody(body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, val)
	if err != nil {
		return errors.New("unmarshalling JSON response failed: " + err.Error())
	}
	return nil
}

// getServerError parses Boruta server response that contains serverError and
// returns an error.
func getServerError(resp *http.Response) error {
	if resp.StatusCode < http.StatusBadRequest {
		return nil
	}
	srvErr := new(util.ServerError)
	switch resp.Header.Get("Content-Type") {
	case contentType:
		if err := bodyJSONUnmarshal(resp.Body, srvErr); err != nil {
			return err
		}
	default:
		msg, err := readBody(resp.Body)
		if err != nil {
			return err
		}
		srvErr.Err = string(msg)
	}
	srvErr.Status = resp.StatusCode
	return srvErr
}

// processResponse is helper function that parses Boruta server response and sets
// returned value or returns serverError. val must be a pointer. In case the body
// was empty (or server returned an error) it will be zeroed - if the val is a
// pointer to ReqInfo then ReqInfo members will be zeroed; to nil a pointer pass
// pointer to pointer to ReqInfo. Function may panic when passed value isn't a pointer.
func processResponse(resp *http.Response, val interface{}) error {
	var v reflect.Value

	if val != nil {
		if reflect.TypeOf(val).Kind() != reflect.Ptr {
			panic("can't set val, please pass appropriate pointer")
		}

		v = reflect.ValueOf(val).Elem()
	}

	setNil := func() {
		if val != nil && !reflect.ValueOf(val).IsNil() {
			v.Set(reflect.Zero(v.Type()))
		}
	}

	switch {
	case resp.StatusCode == http.StatusNoContent:
		setNil()
		return nil
	case resp.StatusCode >= http.StatusBadRequest:
		setNil()
		return getServerError(resp)
	default:
		return bodyJSONUnmarshal(resp.Body, val)
	}
}

// NewRequest creates new Boruta request.
func (client *BorutaClient) NewRequest(caps boruta.Capabilities,
	priority boruta.Priority, owner boruta.UserInfo, validAfter time.Time,
	deadline time.Time) (boruta.ReqID, error) {
	req, err := json.Marshal(&boruta.ReqInfo{
		Priority:   priority,
		Owner:      owner,
		Deadline:   deadline,
		ValidAfter: validAfter,
		Caps:       caps,
	})
	if err != nil {
		return 0, err
	}

	resp, err := http.Post(client.url+"reqs/", contentType, bytes.NewReader(req))
	if err != nil {
		return 0, err
	}
	var reqID util.ReqIDPack
	if err = processResponse(resp, &reqID); err != nil {
		return 0, err
	}
	return reqID.ReqID, nil
}

// CloseRequest closes or cancels Boruta request.
func (client *BorutaClient) CloseRequest(reqID boruta.ReqID) error {
	path := client.url + "reqs/" + strconv.Itoa(int(reqID)) + "/close"
	resp, err := http.Post(path, "", nil)
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// UpdateRequest prepares JSON with fields that should be changed for given
// request ID.
func (client *BorutaClient) UpdateRequest(reqInfo *boruta.ReqInfo) error {
	if reqInfo == nil {
		return errors.New("nil reqInfo passed")
	}
	req, err := json.Marshal(&struct {
		boruta.Priority
		Deadline   time.Time
		ValidAfter time.Time
	}{
		Priority:   reqInfo.Priority,
		Deadline:   reqInfo.Deadline,
		ValidAfter: reqInfo.ValidAfter,
	})
	if err != nil {
		return err
	}
	path := client.url + "reqs/" + strconv.Itoa(int(reqInfo.ID))
	resp, err := http.Post(path, contentType, bytes.NewReader(req))
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// GetRequestInfo queries Boruta server for details about given request ID.
func (client *BorutaClient) GetRequestInfo(reqID boruta.ReqID) (boruta.ReqInfo, error) {
	var reqInfo boruta.ReqInfo
	path := client.url + "reqs/" + strconv.Itoa(int(reqID))
	resp, err := http.Get(path)
	if err != nil {
		return reqInfo, err
	}
	err = processResponse(resp, &reqInfo)
	return reqInfo, err
}

// ListRequests queries Boruta server for list of requests that match given
// filter. Filter may be empty or nil to get list of all requests.
func (client *BorutaClient) ListRequests(filter boruta.ListFilter) ([]boruta.ReqInfo, error) {
	req, err := json.Marshal(filter)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(client.url+"reqs/list", contentType,
		bytes.NewReader(req))
	if err != nil {
		return nil, err
	}
	list := new([]boruta.ReqInfo)
	err = processResponse(resp, list)
	return *list, err
}

// AcquireWorker queries Boruta server for information required to access
// assigned Dryad. Access information may not be available when the call
// is issued because requests need to have assigned worker.
func (client *BorutaClient) AcquireWorker(reqID boruta.ReqID) (boruta.AccessInfo, error) {
	var accInfo boruta.AccessInfo
	path := client.url + "reqs/" + strconv.Itoa(int(reqID)) + "/acquire_worker"
	resp, err := http.Post(path, "", nil)
	if err != nil {
		return accInfo, err
	}
	accInfo2 := new(util.AccessInfo2)
	if err = processResponse(resp, &accInfo2); err != nil {
		return accInfo, err
	}
	block, _ := pem.Decode([]byte(accInfo2.Key))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return accInfo, errors.New("wrong key: " + accInfo2.Key)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return accInfo, err
	}
	accInfo.Addr = accInfo2.Addr
	accInfo.Username = accInfo2.Username
	accInfo.Key = *key
	return accInfo, nil
}

// ProlongAccess requests Boruta server to extend running time of job. User may
// need to call this method multiple times as long as access to Dryad is needed.
// If not called, Boruta server will terminate the tunnel when ReqInfo.Job.Timeout
// passes, and change state of request to CLOSED.
func (client *BorutaClient) ProlongAccess(reqID boruta.ReqID) error {
	path := client.url + "reqs/" + strconv.Itoa(int(reqID)) + "/prolong"
	resp, err := http.Post(path, "", nil)
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// ListWorkers queries Boruta server for list of workers that are in given groups
// and have provided capabilities. Setting both caps and groups to empty or nil
// lists all workers.
func (client *BorutaClient) ListWorkers(groups boruta.Groups,
	caps boruta.Capabilities) ([]boruta.WorkerInfo, error) {
	req, err := json.Marshal(&util.WorkersFilter{
		Groups:       groups,
		Capabilities: caps,
	})
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(client.url+"workers/list", contentType,
		bytes.NewReader(req))
	if err != nil {
		return nil, err
	}
	list := new([]boruta.WorkerInfo)
	err = processResponse(resp, list)
	return *list, err
}

// GetWorkerInfo queries Boruta server for information about worker with given
// UUID.
func (client *BorutaClient) GetWorkerInfo(uuid boruta.WorkerUUID) (boruta.WorkerInfo, error) {
	var info boruta.WorkerInfo
	path := client.url + "workers/" + string(uuid)
	resp, err := http.Get(path)
	if err != nil {
		return info, err
	}
	err = processResponse(resp, &info)
	return info, err
}

// SetState requests Boruta server to change state of worker with provided UUID.
// SetState is intended only for Boruta server administrators.
func (client *BorutaClient) SetState(uuid boruta.WorkerUUID, state boruta.WorkerState) error {
	path := client.url + "workers/" + string(uuid) + "/setstate"
	req, err := json.Marshal(&util.WorkerStatePack{state})
	if err != nil {
		return err
	}
	resp, err := http.Post(path, contentType, bytes.NewReader(req))
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
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
