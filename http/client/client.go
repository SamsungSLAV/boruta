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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	util "github.com/SamsungSLAV/boruta/http"
)

// BorutaClient handles interaction with specified Boruta server.
type BorutaClient struct {
	url string
	boruta.Requests
	boruta.Workers
}

// Version provides information about version of Boruta client package, server, REST API and state
// of the REST API (devel, stable, deprecated).
type Version struct {
	// Client contains version string of client package.
	Client string
	util.BorutaVersion
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

// checkStatus is a helper function that returns an error when HTTP response
// status is different than expected.
func checkStatus(shouldBe int, resp *http.Response) (err error) {
	if resp.StatusCode != shouldBe {
		err = errors.New("bad HTTP status: " + resp.Status)
	}
	return
}

func listInfoFromHeaders(hdr http.Header) (*boruta.ListInfo, error) {
	hdrKeys := [...]string{util.ListTotalItemsHdr, util.ListRemainingItemsHdr}
	var tmp [len(hdrKeys)]uint64

	for i, k := range hdrKeys {
		if _, present := hdr[k]; !present {
			return nil, fmt.Errorf("couldn't find header: %s", k)
		}
		_, err := fmt.Sscanf(hdr.Get(k), "%d", &tmp[i])
		if err != nil {
			return nil, fmt.Errorf("couldn't parse header: %s", err)
		}
	}

	return &boruta.ListInfo{
		TotalItems:     tmp[0],
		RemainingItems: tmp[1],
	}, nil
}

// getHeaders is a helper function that makes HEAD HTTP request for given address,
// checks Status and returns HTTP headers and error.
func getHeaders(url string) (http.Header, error) {
	resp, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	if err = checkStatus(http.StatusNoContent, resp); err != nil {
		return nil, err
	}
	return resp.Header, nil
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
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10) + "/close"
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
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqInfo.ID), 10)
	resp, err := http.Post(path, contentType, bytes.NewReader(req))
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// GetRequestInfo queries Boruta server for details about given request ID.
func (client *BorutaClient) GetRequestInfo(reqID boruta.ReqID) (boruta.ReqInfo, error) {
	var reqInfo boruta.ReqInfo
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10)
	resp, err := http.Get(path)
	if err != nil {
		return reqInfo, err
	}
	err = processResponse(resp, &reqInfo)
	return reqInfo, err
}

// ListRequests queries Boruta server for list of requests that match given filter. Filter may be
// empty or nil to get list of all requests. If sorter is nil then the default sorting is used
// (ascending, by ID). List may be divided into pages. Division is made according to
// boruta.RequestsPaginator. To iterate through whole list, ListRequests should be called repetedly
// with ID changed to ID of last (or first if going backwards) request of slice returned by previous
// call. If paginator is nil then server will use default values. boruta.ListInfo contains
// information how many requests are in filtered list and how many are left till the end of the list.
func (client *BorutaClient) ListRequests(f boruta.ListFilter, s *boruta.SortInfo,
	p *boruta.RequestsPaginator) ([]boruta.ReqInfo, *boruta.ListInfo, error) {

	listSpec := &util.RequestsListSpec{
		Sorter:    s,
		Paginator: p,
	}
	if f != nil {
		rfilter, ok := f.(*filter.Requests)
		if !ok {
			return nil, nil, errors.New("only *filter.Requests type is supported")
		}
		listSpec.Filter = rfilter
	}

	req, err := json.Marshal(listSpec)
	if err != nil {
		return nil, nil, err
	}
	resp, err := http.Post(client.url+"reqs/list", contentType, bytes.NewReader(req))
	if err != nil {
		return nil, nil, err
	}
	info, err := listInfoFromHeaders(resp.Header)
	if err != nil {
		return nil, nil, err
	}
	list := new([]boruta.ReqInfo)
	err = processResponse(resp, list)
	return *list, info, err
}

// AcquireWorker queries Boruta server for information required to access
// assigned Dryad. Access information may not be available when the call
// is issued because requests need to have assigned worker.
func (client *BorutaClient) AcquireWorker(reqID boruta.ReqID) (boruta.AccessInfo, error) {
	var accInfo boruta.AccessInfo
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10) + "/acquire_worker"
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
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10) + "/prolong"
	resp, err := http.Post(path, "", nil)
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// ListWorkers queries Boruta server for list of workers that are in given groups and have provided
// capabilities. Setting both caps and groups to empty or nil lists all workers. If sorter is nil
// then the default sorting is used (ascending, by UUID).
func (client *BorutaClient) ListWorkers(f boruta.ListFilter, s *boruta.SortInfo,
	p *boruta.WorkersPaginator) ([]boruta.WorkerInfo, *boruta.ListInfo, error) {

	listSpec := &util.WorkersListSpec{
		Sorter:    s,
		Paginator: p,
	}
	if f != nil {
		wfilter, ok := f.(*filter.Workers)
		if !ok {
			return nil, nil, errors.New("only *filter.Workers type is supported")
		}
		listSpec.Filter = wfilter
	}
	req, err := json.Marshal(listSpec)
	if err != nil {
		return nil, nil, err
	}
	resp, err := http.Post(client.url+"workers/list", contentType, bytes.NewReader(req))
	if err != nil {
		return nil, nil, err
	}
	info, err := listInfoFromHeaders(resp.Header)
	if err != nil {
		return nil, nil, err
	}
	list := new([]boruta.WorkerInfo)
	err = processResponse(resp, list)
	return *list, info, err
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
	req, err := json.Marshal(&util.WorkerStatePack{WorkerState: state})
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
func (client *BorutaClient) SetGroups(uuid boruta.WorkerUUID, groups boruta.Groups) error {
	path := client.url + "workers/" + string(uuid) + "/setgroups"
	req, err := json.Marshal(groups)
	if err != nil {
		return err
	}
	resp, err := http.Post(path, contentType, bytes.NewReader(req))
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// Deregister requests Boruta server to deregister worker with provided UUID.
// Deregister is intended only for Boruta server administrators.
func (client *BorutaClient) Deregister(uuid boruta.WorkerUUID) error {
	path := client.url + "workers/" + string(uuid) + "/deregister"
	resp, err := http.Post(path, "", nil)
	if err != nil {
		return err
	}
	return processResponse(resp, nil)
}

// GetRequestState is convenient way to check state of a request with given reqID.
// When error occurs then returned boruta.ReqState will make no sense. Developer
// should always check for an error before proceeding with actions dependent on
// request state.
func (client *BorutaClient) GetRequestState(reqID boruta.ReqID) (boruta.ReqState, error) {
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10)
	headers, err := getHeaders(path)
	if err != nil {
		return boruta.FAILED, err
	}
	return boruta.ReqState(headers.Get("Boruta-Request-State")), nil
}

// GetWorkerState is convenient way to check state of a worker with given UUID.
func (client *BorutaClient) GetWorkerState(uuid boruta.WorkerUUID) (boruta.WorkerState, error) {
	path := client.url + "workers/" + string(uuid)
	headers, err := getHeaders(path)
	if err != nil {
		return boruta.FAIL, err
	}
	return boruta.WorkerState(headers.Get("Boruta-Worker-State")), nil
}

// GetJobTimeout is convenient way to check when Job of a request with given
// reqID will timeout. The request must be in INPROGRESS state.
func (client *BorutaClient) GetJobTimeout(reqID boruta.ReqID) (time.Time, error) {
	var t time.Time
	path := client.url + "reqs/" + strconv.FormatUint(uint64(reqID), 10)
	headers, err := getHeaders(path)
	if err != nil {
		return t, err
	}
	if boruta.ReqState(headers.Get("Boruta-Request-State")) != boruta.INPROGRESS {
		return t, errors.New(`request must be in "IN PROGRESS" state`)
	}
	return time.Parse(util.DateFormat, headers.Get("Boruta-Job-Timeout"))
}

// Version provides information about version of Boruta client package, server, REST API version
// and REST API state.
func (client *BorutaClient) Version() (*Version, error) {
	path := client.url + "/version"
	headers, err := getHeaders(path)
	if err != nil {
		return nil, err
	}
	return &Version{
		Client: boruta.Version,
		BorutaVersion: util.BorutaVersion{
			Server: headers.Get(util.ServerVersionHdr),
			API:    headers.Get(util.APIVersionHdr),
			State:  headers.Get(util.APIStateHdr),
		},
	}, nil
}
