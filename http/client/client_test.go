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

package client

import (
	"testing"
	"time"

	. "git.tizen.org/tools/boruta"
	util "git.tizen.org/tools/boruta/http"
	"github.com/stretchr/testify/assert"
)

const url = "http://localhost:8080"

func initTest(t *testing.T) (*assert.Assertions, *BorutaClient) {
	return assert.New(t), NewBorutaClient(url)
}

func TestNewBorutaClient(t *testing.T) {
	assert, client := initTest(t)

	assert.NotNil(client)
	assert.Equal(url+"/api/v1/", client.url)
}

func TestNewRequest(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	reqID, err := client.NewRequest(make(Capabilities), Priority(0),
		UserInfo{}, time.Now(), time.Now())
	assert.Zero(reqID)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestCloseRequest(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.CloseRequest(ReqID(0))
	assert.Equal(util.ErrNotImplemented, err)
}

func TestUpdateRequest(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.UpdateRequest(nil)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestGetRequestInfo(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	reqInfo, err := client.GetRequestInfo(ReqID(0))
	assert.Nil(reqInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestListRequests(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	reqInfo, err := client.ListRequests(nil)
	assert.Nil(reqInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestAcquireWorker(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	accessInfo, err := client.AcquireWorker(ReqID(0))
	assert.Nil(accessInfo)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestProlongAccess(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.ProlongAccess(ReqID(0))
	assert.Equal(util.ErrNotImplemented, err)
}

func TestListWorkers(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	list, err := client.ListWorkers(nil, nil)
	assert.Nil(list)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestGetWorkerInfo(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	info, err := client.GetWorkerInfo(WorkerUUID(""))
	assert.Zero(info)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestSetState(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.SetState(WorkerUUID(""), FAIL)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestSetGroups(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.SetGroups(WorkerUUID(""), nil)
	assert.Equal(util.ErrNotImplemented, err)
}

func TestDeregister(t *testing.T) {
	assert, client := initTest(t)
	assert.NotNil(client)

	err := client.Deregister(WorkerUUID(""))
	assert.Equal(util.ErrNotImplemented, err)
}
