/*
 *  Copyright (c) 2017 Samsung Electronics Co., Ltd All Rights Reserved
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
	"strconv"
	"testing"
	"time"

	. "git.tizen.org/tools/boruta"
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

func initTest(t *testing.T) (*assert.Assertions, *ReqsCollection) {
	return assert.New(t), NewRequestQueue()
}

func TestNewRequestQueue(t *testing.T) {
	assert, q := initTest(t)

	assert.Zero(len(q.requests))
	assert.NotNil(q.queue)
	assert.Zero(q.queue.length)
}

func TestNewRequest(t *testing.T) {
	assert, rqueue := initTest(t)

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
	res := rqueue.requests[reqid]
	assert.True(start.Before(res.ValidAfter) && stop.After(res.ValidAfter))
	start = start.AddDate(0, 1, 0)
	stop = stop.AddDate(0, 1, 0)
	assert.True(start.Before(res.Deadline) && stop.After(res.Deadline))
	assert.EqualValues(2, rqueue.queue.length)
}

func TestCloseRequest(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req

	// Add valid request to the queue.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	// Cancel previously added request.
	assert.EqualValues(1, rqueue.queue.length)
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	assert.Equal(ReqState(CANCEL), rqueue.requests[reqid].State)
	assert.Zero(rqueue.queue.length)

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
	rqueue.requests[reqid].State = INPROGRESS
	rqueue.queue.removeRequest(&reqinfo)
	// Close request.
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	assert.EqualValues(2, len(rqueue.requests))
	assert.Equal(ReqState(DONE), rqueue.requests[reqid].State)

	// Simulation for the rest of states.
	states := [...]ReqState{INVALID, CANCEL, TIMEOUT, DONE, FAILED}
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(3, reqid)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	rqueue.queue.removeRequest(&reqinfo)
	for i := range states {
		rqueue.requests[reqid].State = states[i]
		err = rqueue.CloseRequest(reqid)
		assert.EqualValues(ErrModificationForbidden, err)
	}

	assert.EqualValues(3, len(rqueue.requests))
	assert.EqualValues(0, rqueue.queue.length)
}

func TestSetRequestPriority(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	legalPrio := req.Priority + 1
	illegalPrio := 2 * LoPrio

	// Change priority of the request to the same value.
	err = rqueue.SetRequestPriority(reqid, req.Priority)
	assert.Nil(err)
	reqinfo, err := rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(req.Priority, reqinfo.Priority)

	// Change priority of the request.
	err = rqueue.SetRequestPriority(reqid, legalPrio)
	assert.Nil(err)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(legalPrio, reqinfo.Priority)

	// Try to change priority of request that doesn't exist.
	err = rqueue.SetRequestPriority(req.ID+1, legalPrio)
	assert.Equal(NotFoundError("Request"), err)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(legalPrio, reqinfo.Priority)

	// Try to change priority of request to invalid value.
	err = rqueue.SetRequestPriority(reqid, illegalPrio)
	assert.Equal(ErrPriority, err)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(legalPrio, reqinfo.Priority)

	// Try to change priority of request which is in state that forbid changes.
	rqueue.requests[reqid].State = INVALID
	err = rqueue.SetRequestPriority(reqid, legalPrio)
	assert.EqualValues(ErrModificationForbidden, err)
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	assert.Equal(legalPrio, reqinfo.Priority)
}

func TestSetRequestValidAfter(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	// Set legal ValidAfter value.
	err = rqueue.SetRequestValidAfter(reqid, tomorrow)
	assert.Nil(err)
	assert.Equal(tomorrow, rqueue.requests[reqid].ValidAfter)

	// Try to set ValidAfter for non-existent request.
	err = rqueue.SetRequestValidAfter(ReqID(2), tomorrow)
	assert.Equal(NotFoundError("Request"), err)

	// Try to set ValidAfter newer then Deadline.
	rqueue.requests[reqid].Deadline = now
	err = rqueue.SetRequestValidAfter(reqid, tomorrow)
	assert.Equal(ErrInvalidTimeRange, err)

	// Try to set ValidAfter for request which cannot be modified.
	rqueue.requests[reqid].State = INVALID
	err = rqueue.SetRequestValidAfter(reqid, yesterday)
	assert.EqualValues(ErrModificationForbidden, err)
}

func TestSetRequestDeadline(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	// Set legal Deadline value.
	dayAfter := tomorrow.AddDate(0, 0, 1).UTC()
	err = rqueue.SetRequestDeadline(reqid, dayAfter)
	assert.Nil(err)
	assert.Equal(dayAfter, rqueue.requests[reqid].Deadline)

	// Try to set Deadline for non-existent request.
	err = rqueue.SetRequestDeadline(ReqID(2), tomorrow)
	assert.Equal(NotFoundError("Request"), err)

	// Try to set Deadline in the past.
	err = rqueue.SetRequestDeadline(reqid, yesterday)
	assert.Equal(ErrDeadlineInThePast, err)

	// Try to set Deadline before ValidAfter.
	rqueue.requests[reqid].ValidAfter = dayAfter
	err = rqueue.SetRequestDeadline(reqid, tomorrow)
	assert.Equal(ErrInvalidTimeRange, err)

	// Try to set Deadline for request which cannot be modified.
	rqueue.requests[reqid].State = INVALID
	err = rqueue.SetRequestDeadline(reqid, tomorrow)
	assert.EqualValues(ErrModificationForbidden, err)
}

func TestGetRequestInfo(t *testing.T) {
	assert, rqueue := initTest(t)
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

	priority := strconv.FormatUint(uint64(req.Priority), 10)
	if filter.priority != "" && priority != filter.priority {
		return false
	}

	return true
}

func TestListRequests(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req
	const reqsCnt = 4

	// Add few requests.
	reqs := make(map[ReqID]bool, reqsCnt)
	noReqs := make(map[ReqID]bool)
	for i := 0; i < reqsCnt; i++ {
		reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
		assert.Nil(err)
		if i%2 == 1 {
			rqueue.requests[reqid].Priority++
		}
		if i > 1 {
			rqueue.requests[reqid].State = DONE
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
				priority: strconv.FormatUint(uint64(req.Priority), 10),
			},
			result: map[ReqID]bool{ReqID(1): true},
		},
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: strconv.FormatUint(uint64(req.Priority+1), 10),
			},
			result: map[ReqID]bool{ReqID(2): true},
		},
		{
			filter: reqFilter{
				state:    string(DONE),
				priority: strconv.FormatUint(uint64(req.Priority), 10),
			},
			result: map[ReqID]bool{ReqID(3): true},
		},
		{
			filter: reqFilter{
				state:    string(DONE),
				priority: strconv.FormatUint(uint64(req.Priority+1), 10),
			},
			result: map[ReqID]bool{ReqID(4): true},
		},
		{
			filter: reqFilter{
				state:    "",
				priority: strconv.FormatUint(uint64(req.Priority), 10),
			},
			result: map[ReqID]bool{ReqID(1): true, ReqID(3): true},
		},
		{
			filter: reqFilter{
				state:    "",
				priority: strconv.FormatUint(uint64(req.Priority+1), 10),
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
				priority: strconv.FormatUint(uint64(notFoundPrio), 10),
			},
			result: noReqs,
		},
		{
			filter: reqFilter{
				state:    string(WAIT),
				priority: strconv.FormatUint(uint64(notFoundPrio), 10),
			},
			result: noReqs,
		},
		{
			filter: reqFilter{
				state:    string(notFoundState),
				priority: strconv.FormatUint(uint64(req.Priority), 10),
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
	resp, err := rqueue.ListRequests(nil)
	assert.Nil(err)
	checkReqs(reqs, resp)
}

func TestAcquireWorker(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req
	empty := AccessInfo{}

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	states := [...]ReqState{WAIT, INVALID, CANCEL, TIMEOUT, DONE, FAILED, INPROGRESS}
	for _, state := range states {
		rqueue.requests[reqid].State = state
		ainfo, err := rqueue.AcquireWorker(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
		assert.Equal(empty, ainfo)
	}

	// Try to acquire worker for non-existing request.
	ainfo, err := rqueue.AcquireWorker(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)
	assert.Equal(empty, ainfo)

	// AcquireWorker to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	rqueue.requests[reqid].Job = new(JobInfo)
	ainfo, err = rqueue.AcquireWorker(reqid)
	assert.Nil(err)
	assert.Equal(empty, ainfo)
}

func TestProlongAccess(t *testing.T) {
	assert, rqueue := initTest(t)
	req := requestsTests[0].req

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	states := [...]ReqState{WAIT, INVALID, CANCEL, TIMEOUT, DONE, FAILED, INPROGRESS}
	for _, state := range states {
		rqueue.requests[reqid].State = state
		err = rqueue.ProlongAccess(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
	}

	// Try to prolong access of job for non-existing request.
	err = rqueue.ProlongAccess(ReqID(2))
	assert.Equal(NotFoundError("Request"), err)

	// ProlongAccess to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	rqueue.requests[reqid].Job = new(JobInfo)
	err = rqueue.ProlongAccess(reqid)
	assert.Nil(err)
}
