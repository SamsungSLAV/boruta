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

package requests

import (
	"errors"
	"net"
	"testing"
	"time"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	owner     UserInfo
	job       JobInfo
	zeroTime  time.Time
	caps      = make(Capabilities)
	now       = time.Now().UTC()
	yesterday = now.AddDate(0, 0, -1).UTC()
	tomorrow  = now.AddDate(0, 0, 1).UTC()
)

var requestsTests = [...]struct {
	req ReqInfo
	err error
}{
	{
		// valid request
		req: ReqInfo{ReqID(1), Priority((HiPrio + LoPrio) / 2), owner, tomorrow, now, WAIT, &job, caps},
		err: nil,
	},
	{
		// request with invalid priority
		req: ReqInfo{ReqID(0), Priority(LoPrio + 1), owner, tomorrow, now, WAIT, &job, caps},
		err: ErrPriority,
	},
	{
		// request with ValidAfter date newer then Deadline
		req: ReqInfo{ReqID(0), Priority((HiPrio + LoPrio) / 2), owner, now.Add(time.Hour), tomorrow, WAIT, &job, caps},
		err: ErrInvalidTimeRange,
	},
	{
		// request with Deadline set in the past.
		req: ReqInfo{ReqID(0), Priority((HiPrio + LoPrio) / 2), owner, yesterday, now, WAIT, &job, caps},
		err: ErrDeadlineInThePast,
	},
}

func initTest(t *testing.T) (*assert.Assertions, *ReqsCollection, *gomock.Controller, *MockJobsManager) {
	ctrl := gomock.NewController(t)
	wm := NewMockWorkersManager(ctrl)
	jm := NewMockJobsManager(ctrl)
	testErr := errors.New("Test Error")
	wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(WorkerUUID(""), testErr).AnyTimes()
	wm.EXPECT().SetChangeListener(gomock.Any())
	return assert.New(t), NewRequestQueue(wm, jm), ctrl, jm
}

func finiTest(rqueue *ReqsCollection, ctrl *gomock.Controller) {
	rqueue.Finish()
	ctrl.Finish()
}

func TestNewRequestQueue(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)

	rqueue.mutex.RLock()
	defer rqueue.mutex.RUnlock()
	assert.Zero(len(rqueue.requests))
	assert.NotNil(rqueue.queue)
	assert.Zero(rqueue.queue.length)
}

func TestNewRequest(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)

	for _, test := range requestsTests {
		reqid, err := rqueue.NewRequest(test.req.Caps, test.req.Priority,
			test.req.Owner, test.req.ValidAfter, test.req.Deadline)
		assert.Equal(test.req.ID, reqid)
		assert.Equal(test.err, err)
	}

	req := requestsTests[0].req
	req.Deadline = zeroTime
	req.ValidAfter = zeroTime
	start := time.Now()
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner,
		req.ValidAfter, req.Deadline)
	stop := time.Now()
	assert.Nil(err)
	rqueue.mutex.RLock()
	defer rqueue.mutex.RUnlock()
	res := rqueue.requests[reqid]
	assert.True(start.Before(res.ValidAfter) && stop.After(res.ValidAfter))
	start = start.AddDate(0, 1, 0)
	stop = stop.AddDate(0, 1, 0)
	assert.True(start.Before(res.Deadline) && stop.After(res.Deadline))
	assert.EqualValues(2, rqueue.queue.length)
}

func TestCloseRequest(t *testing.T) {
	assert, rqueue, ctrl, jm := initTest(t)
	defer finiTest(rqueue, ctrl)

	req := requestsTests[0].req
	jobInfo := JobInfo{
		WorkerUUID: "Test WorkerUUID",
	}

	// Add valid request to the queue.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	// Cancel previously added request.
	rqueue.mutex.RLock()
	assert.EqualValues(1, rqueue.queue.length)
	rqueue.mutex.RUnlock()
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(ReqState(CANCEL), rqueue.requests[reqid].State)
	assert.Zero(rqueue.queue.length)
	rqueue.mutex.RUnlock()

	// Try to close non-existent request.
	err = rqueue.CloseRequest(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)

	// Add another valid request.
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(2, reqid)
	// Simulate situation where request was assigned a worker and job has begun.
	reqinfo, err := rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	rqueue.mutex.Lock()
	rqueue.requests[reqid].State = INPROGRESS
	rqueue.requests[reqid].Job = &jobInfo
	rqueue.queue.removeRequest(&reqinfo)
	rqueue.mutex.Unlock()
	// Close request.
	gomock.InOrder(
		jm.EXPECT().Finish(jobInfo.WorkerUUID),
	)
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.EqualValues(2, len(rqueue.requests))
	assert.Equal(ReqState(DONE), rqueue.requests[reqid].State)
	rqueue.mutex.RUnlock()

	// Simulation for the rest of states.
	states := [...]ReqState{INVALID, CANCEL, TIMEOUT, DONE, FAILED}
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(3, reqid)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	rqueue.mutex.Lock()
	rqueue.queue.removeRequest(&reqinfo)
	rqueue.mutex.Unlock()
	for i := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = states[i]
		rqueue.mutex.Unlock()
		err = rqueue.CloseRequest(reqid)
		assert.EqualValues(ErrModificationForbidden, err)
	}

	rqueue.mutex.RLock()
	defer rqueue.mutex.RUnlock()
	assert.EqualValues(3, len(rqueue.requests))
	assert.EqualValues(0, rqueue.queue.length)
}

func TestUpdateRequest(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	tmp := requestsTests[0].req

	// Add valid request.
	reqid, err := rqueue.NewRequest(tmp.Caps, tmp.Priority, tmp.Owner, tmp.ValidAfter, tmp.Deadline)
	assert.Nil(err)
	rqueue.mutex.RLock()
	req := rqueue.requests[reqid]
	rqueue.mutex.RUnlock()
	reqBefore, err := rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	reqUpdate := new(ReqInfo)
	rqueue.mutex.RLock()
	*reqUpdate = *req
	rqueue.mutex.RUnlock()

	// Check noop.
	err = rqueue.UpdateRequest(nil)
	assert.Nil(err)
	reqUpdate.ValidAfter = zeroTime
	reqUpdate.Deadline = zeroTime
	reqUpdate.Priority = Priority(0)
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(req, &reqBefore)
	// Check request that doesn't exist.
	*reqUpdate = *req
	rqueue.mutex.RUnlock()
	reqUpdate.ID++
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(NotFoundError("Request"), err)
	rqueue.mutex.RLock()
	reqUpdate.ID = req.ID
	// Change Priority only.
	reqUpdate.Priority = req.Priority - 1
	rqueue.mutex.RUnlock()
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(reqUpdate.Priority, req.Priority)
	rqueue.mutex.RUnlock()
	// Change ValidAfter only.
	reqUpdate.ValidAfter = yesterday
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(reqUpdate.ValidAfter, req.ValidAfter)
	rqueue.mutex.RUnlock()
	// Change Deadline only.
	reqUpdate.Deadline = tomorrow.AddDate(0, 0, 1).UTC()
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(reqUpdate.Deadline, req.Deadline)
	rqueue.mutex.RUnlock()
	// Change Priority, ValidAfter and Deadline.
	reqUpdate.Deadline = tomorrow
	reqUpdate.ValidAfter = time.Now().Add(time.Hour)
	reqUpdate.Priority = LoPrio
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(reqUpdate, req)
	rqueue.mutex.RUnlock()
	// Change values to the same ones that are already set.
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(reqUpdate, req)
	rqueue.mutex.RUnlock()
	// Change Priority to illegal value.
	reqUpdate.Priority = LoPrio + 1
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(ErrPriority, err)
	rqueue.mutex.RLock()
	reqUpdate.Priority = req.Priority
	rqueue.mutex.RUnlock()
	//Change Deadline to illegal value.
	reqUpdate.Deadline = yesterday
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(ErrDeadlineInThePast, err)
	reqUpdate.Deadline = time.Now().Add(time.Minute)
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(ErrInvalidTimeRange, err)
	// Change ValidAfer to illegal value.
	rqueue.mutex.RLock()
	reqUpdate.ValidAfter = req.Deadline.Add(time.Hour)
	rqueue.mutex.RUnlock()
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(ErrInvalidTimeRange, err)
	// Try to change values for other changes.
	states := [...]ReqState{INVALID, CANCEL, TIMEOUT, DONE, FAILED, INPROGRESS}
	for _, state := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = state
		rqueue.mutex.Unlock()
		err = rqueue.UpdateRequest(reqUpdate)
		assert.Equal(ErrModificationForbidden, err)
	}
}

func TestGetRequestInfo(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req
	req.Job = nil
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	// Get request information for existing request.
	reqUpdate, err := rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(req, reqUpdate)

	// Try to get information for non-existent request.
	req3, err := rqueue.GetRequestInfo(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)
	assert.Zero(req3)
}

type reqFilter struct {
	state    string
	priority string
}

func (filter *reqFilter) Match(req *ReqInfo) bool {
	if req == nil {
		return false
	}

	if filter.state != "" && string(req.State) != filter.state {
		return false
	}

	priority := req.Priority.String()
	if filter.priority != "" && priority != filter.priority {
		return false
	}

	return true
}

func TestListRequests(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req
	const reqsCnt = 4

	// Add few requests.
	reqs := make(map[ReqID]bool, reqsCnt)
	noReqs := make(map[ReqID]bool)
	for i := 0; i < reqsCnt; i++ {
		reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
		assert.Nil(err)
		if i%2 == 1 {
			rqueue.mutex.Lock()
			rqueue.requests[reqid].Priority++
			rqueue.mutex.Unlock()
		}
		if i > 1 {
			rqueue.mutex.Lock()
			rqueue.requests[reqid].State = DONE
			rqueue.mutex.Unlock()
		}
		reqs[reqid] = true
	}

	notFoundPrio := req.Priority - 1
	notFoundState := INVALID
	var filterTests = [...]struct {
		filter reqFilter
		result map[ReqID]bool
	}{
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: req.Priority.String(),
			},
			result: map[ReqID]bool{ReqID(1): true},
		},
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: (req.Priority + 1).String(),
			},
			result: map[ReqID]bool{ReqID(2): true},
		},
		{
			filter: reqFilter{
				state:    string(DONE),
				priority: req.Priority.String(),
			},
			result: map[ReqID]bool{ReqID(3): true},
		},
		{
			filter: reqFilter{
				state:    string(DONE),
				priority: (req.Priority + 1).String(),
			},
			result: map[ReqID]bool{ReqID(4): true},
		},
		{
			filter: reqFilter{
				state:    "",
				priority: req.Priority.String(),
			},
			result: map[ReqID]bool{ReqID(1): true, ReqID(3): true},
		},
		{
			filter: reqFilter{
				state:    "",
				priority: (req.Priority + 1).String(),
			},
			result: map[ReqID]bool{ReqID(2): true, ReqID(4): true},
		},
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: "",
			},
			result: map[ReqID]bool{ReqID(1): true, ReqID(2): true},
		},
		{
			filter: reqFilter{
				state:    string(DONE),
				priority: "",
			},
			result: map[ReqID]bool{ReqID(3): true, ReqID(4): true},
		},
		{
			filter: reqFilter{
				state:    "",
				priority: "",
			},
			result: reqs,
		},
		{
			filter: reqFilter{
				state:    string(notFoundState),
				priority: notFoundPrio.String(),
			},
			result: noReqs,
		},
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: notFoundPrio.String(),
			},
			result: noReqs,
		},
		{
			filter: reqFilter{
				state:    string(notFoundState),
				priority: req.Priority.String(),
			},
			result: noReqs,
		},
	}

	checkReqs := func(reqs map[ReqID]bool, resp []ReqInfo) {
		assert.Equal(len(reqs), len(resp))
		for _, req := range resp {
			assert.True(reqs[req.ID])
		}
	}

	for _, test := range filterTests {
		resp, err := rqueue.ListRequests(&test.filter)
		assert.Nil(err)
		checkReqs(test.result, resp)
	}

	// Nil filter should return all requests.

	// Nil interface.
	resp, err := rqueue.ListRequests(nil)
	assert.Nil(err)
	checkReqs(reqs, resp)
	var flt *reqFilter
	// Concrete type is nil but interface isn't nil.
	resp, err = rqueue.ListRequests(flt)
	assert.Nil(err)
	checkReqs(reqs, resp)
}

func TestAcquireWorker(t *testing.T) {
	assert, rqueue, ctrl, jm := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req
	empty := AccessInfo{}
	testErr := errors.New("Test Error")

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	states := [...]ReqState{WAIT, INVALID, CANCEL, TIMEOUT, DONE, FAILED, INPROGRESS}
	for _, state := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = state
		rqueue.mutex.Unlock()
		ainfo, err := rqueue.AcquireWorker(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
		assert.Equal(empty, ainfo)
	}

	// Try to acquire worker for non-existing request.
	ainfo, err := rqueue.AcquireWorker(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)
	assert.Equal(empty, ainfo)

	// Try to acquire worker when jobs.Get() fails.
	jobInfo := JobInfo{
		WorkerUUID: "Test WorkerUUID",
	}
	rqueue.mutex.Lock()
	rqueue.requests[reqid].Job = &jobInfo
	rqueue.mutex.Unlock()
	ignoredJob := &workers.Job{Req: ReqID(0xBAD)}
	jm.EXPECT().Get(jobInfo.WorkerUUID).Return(ignoredJob, testErr)
	ainfo, err = rqueue.AcquireWorker(reqid)
	assert.Equal(testErr, err)
	assert.Equal(empty, ainfo)

	// AcquireWorker to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	job := &workers.Job{
		Access: AccessInfo{Addr: &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4)}},
	}
	rqueue.mutex.Lock()
	rqueue.requests[reqid].Job = &jobInfo
	rqueue.mutex.Unlock()
	jm.EXPECT().Get(jobInfo.WorkerUUID).Return(job, nil)
	ainfo, err = rqueue.AcquireWorker(reqid)
	assert.Nil(err)
	assert.Equal(job.Access, ainfo)
}

func TestProlongAccess(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	states := [...]ReqState{WAIT, INVALID, CANCEL, TIMEOUT, DONE, FAILED, INPROGRESS}
	for _, state := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = state
		rqueue.mutex.Unlock()
		err = rqueue.ProlongAccess(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
	}

	// Try to prolong access of job for non-existing request.
	err = rqueue.ProlongAccess(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)

	// ProlongAccess to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	rqueue.mutex.Lock()
	rqueue.requests[reqid].Job = new(JobInfo)
	rqueue.mutex.Unlock()
	err = rqueue.ProlongAccess(reqid)
	assert.Nil(err)
}
