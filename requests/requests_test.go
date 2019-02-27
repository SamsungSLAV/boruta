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

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	"github.com/SamsungSLAV/boruta/workers"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	owner     boruta.UserInfo
	job       boruta.JobInfo
	zeroTime  time.Time
	caps      = make(boruta.Capabilities)
	now       = time.Now().UTC()
	lastWeek  = now.AddDate(0, 0, -7).UTC()
	yesterday = now.AddDate(0, 0, -1).UTC()
	tomorrow  = now.AddDate(0, 0, 1).UTC()
	nextWeek  = now.AddDate(0, 0, 7).UTC()
)

var requestsTests = [...]struct {
	req boruta.ReqInfo
	err error
}{
	{
		// valid request
		req: boruta.ReqInfo{
			ID:         boruta.ReqID(1),
			Priority:   boruta.Priority((boruta.HiPrio + boruta.LoPrio) / 2),
			Owner:      owner,
			Deadline:   tomorrow,
			ValidAfter: now,
			State:      boruta.WAIT,
			Job:        &job,
			Caps:       caps,
		},
		err: nil,
	},
	{
		// request with invalid priority
		req: boruta.ReqInfo{
			ID:         boruta.ReqID(0),
			Priority:   boruta.Priority(boruta.LoPrio + 1),
			Owner:      owner,
			Deadline:   tomorrow,
			ValidAfter: now,
			State:      boruta.WAIT,
			Job:        &job,
			Caps:       caps,
		},
		err: ErrPriority,
	},
	{
		// request with ValidAfter date newer then Deadline
		req: boruta.ReqInfo{
			ID:         boruta.ReqID(0),
			Priority:   boruta.Priority((boruta.HiPrio + boruta.LoPrio) / 2),
			Owner:      owner,
			Deadline:   now.Add(time.Hour),
			ValidAfter: tomorrow,
			State:      boruta.WAIT,
			Job:        &job,
			Caps:       caps,
		},
		err: ErrInvalidTimeRange,
	},
	{
		// request with Deadline set in the past.
		req: boruta.ReqInfo{
			ID:         boruta.ReqID(0),
			Priority:   boruta.Priority((boruta.HiPrio + boruta.LoPrio) / 2),
			Owner:      owner,
			Deadline:   yesterday,
			ValidAfter: now,
			State:      boruta.WAIT,
			Job:        &job,
			Caps:       caps,
		},
		err: ErrDeadlineInThePast,
	},
}

func initTest(t *testing.T) (*assert.Assertions, *ReqsCollection, *gomock.Controller, *MockJobsManager) {
	ctrl := gomock.NewController(t)
	wm := NewMockWorkersManager(ctrl)
	jm := NewMockJobsManager(ctrl)
	testErr := errors.New("Test Error")
	wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""),
		testErr).AnyTimes()
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
	jobInfo := boruta.JobInfo{
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
	assert.Equal(boruta.ReqState(boruta.CANCEL), rqueue.requests[reqid].State)
	assert.Zero(rqueue.queue.length)
	rqueue.mutex.RUnlock()

	// Try to close non-existent request.
	err = rqueue.CloseRequest(boruta.ReqID(2))
	assert.Equal(boruta.NotFoundError("Request"), err)

	// Add another valid request.
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(2, reqid)
	// Simulate situation where request was assigned a worker and job has begun.
	reqinfo, err := rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	rqueue.mutex.Lock()
	rqueue.requests[reqid].State = boruta.INPROGRESS
	rqueue.requests[reqid].Job = &jobInfo
	rqueue.queue.removeRequest(&reqinfo)
	rqueue.mutex.Unlock()
	// Close request.
	gomock.InOrder(
		jm.EXPECT().Finish(jobInfo.WorkerUUID, true),
	)
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.EqualValues(2, len(rqueue.requests))
	assert.Equal(boruta.ReqState(boruta.DONE), rqueue.requests[reqid].State)
	rqueue.mutex.RUnlock()

	// Add another another valid request.
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(3, reqid)
	// Simulate situation where request is in PROGRESS state, but no job for it exists.
	reqinfo, err = rqueue.GetRequestInfo(reqid)
	assert.Nil(err)
	rqueue.mutex.Lock()
	rqueue.requests[reqid].State = boruta.INPROGRESS
	rqueue.requests[reqid].Job = nil
	rqueue.queue.removeRequest(&reqinfo)
	rqueue.mutex.Unlock()
	// Close request.
	err = rqueue.CloseRequest(reqid)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.EqualValues(3, len(rqueue.requests))
	assert.Equal(boruta.ReqState(boruta.DONE), rqueue.requests[reqid].State)
	rqueue.mutex.RUnlock()

	// Simulation for the rest of states.
	states := [...]boruta.ReqState{boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT, boruta.DONE,
		boruta.FAILED}
	reqid, err = rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)
	assert.EqualValues(4, reqid)
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
	assert.EqualValues(4, len(rqueue.requests))
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
	reqUpdate := new(boruta.ReqInfo)
	rqueue.mutex.RLock()
	*reqUpdate = *req
	rqueue.mutex.RUnlock()

	// Check noop.
	err = rqueue.UpdateRequest(nil)
	assert.Nil(err)
	reqUpdate.ValidAfter = zeroTime
	reqUpdate.Deadline = zeroTime
	reqUpdate.Priority = boruta.Priority(0)
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Nil(err)
	rqueue.mutex.RLock()
	assert.Equal(req, &reqBefore)
	// Check request that doesn't exist.
	*reqUpdate = *req
	rqueue.mutex.RUnlock()
	reqUpdate.ID++
	err = rqueue.UpdateRequest(reqUpdate)
	assert.Equal(boruta.NotFoundError("Request"), err)
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
	reqUpdate.Priority = boruta.LoPrio
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
	reqUpdate.Priority = boruta.LoPrio + 1
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
	states := [...]boruta.ReqState{boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT, boruta.DONE,
		boruta.FAILED, boruta.INPROGRESS}
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
	req3, err := rqueue.GetRequestInfo(boruta.ReqID(2))
	assert.Equal(boruta.NotFoundError("Request"), err)
	assert.Zero(req3)
}

func TestListRequests(t *testing.T) {
	_, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req
	const reqsCnt = 16
	si := &boruta.SortInfo{
		Item:  "id",
		Order: boruta.SortOrderDesc,
	}

	getResults := func(ids ...int) (res []*boruta.ReqInfo) {
		res = make([]*boruta.ReqInfo, len(ids))
		for idx, id := range ids {
			res[idx] = rqueue.requests[boruta.ReqID(id)]
		}
		return
	}

	// Add few requests.
	for i := 0; i < reqsCnt; i++ {
		reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter,
			req.Deadline)
		if err != nil {
			t.Fatal("unable to create new request:", err)
		}
		if i%2 == 1 {
			rqueue.mutex.Lock()
			rqueue.requests[reqid].Priority++
			rqueue.mutex.Unlock()
		}
		if i > 1 {
			rqueue.mutex.Lock()
			rqueue.requests[reqid].State = boruta.DONE
			rqueue.mutex.Unlock()
		}
	}

	notFoundPrio := req.Priority - 1
	notFoundState := boruta.INVALID
	notFoundReqID := boruta.ReqID(15190)
	var filterTests = [...]struct {
		f      *filter.Requests
		s      *boruta.SortInfo
		p      *boruta.RequestsPaginator
		result []*boruta.ReqInfo
		info   *boruta.ListInfo
		err    error
		name   string
	}{
		{
			f: &filter.Requests{
				States:     []boruta.ReqState{boruta.WAIT},
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      si,
			p:      nil,
			result: getResults(1),
			info: &boruta.ListInfo{
				TotalItems:     1,
				RemainingItems: 0,
			},
			name: "filter by WAIT state and higher priority",
		},
		{
			f: &filter.Requests{
				States:     []boruta.ReqState{boruta.WAIT},
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s:      si,
			p:      nil,
			result: getResults(2),
			info: &boruta.ListInfo{
				TotalItems:     1,
				RemainingItems: 0,
			},
			name: "filter by WAIT state and lower priority",
		},
		{
			f: &filter.Requests{
				States:     []boruta.ReqState{boruta.DONE},
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      nil,
			p:      nil,
			result: getResults(3, 5, 7, 9, 11, 13, 15),
			info: &boruta.ListInfo{
				TotalItems:     7,
				RemainingItems: 0,
			},
			name: "filter by DONE state and higher priority",
		},
		{
			f: &filter.Requests{
				States:     []boruta.ReqState{boruta.DONE},
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s:      nil,
			p:      nil,
			result: getResults(4, 6, 8, 10, 12, 14, 16),
			info: &boruta.ListInfo{
				TotalItems:     7,
				RemainingItems: 0,
			},
			name: "filter by DONE state and lower priority",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{0, 3, 6, 9, 12, 15},
				States:     []boruta.ReqState{boruta.DONE},
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      nil,
			p:      nil,
			result: getResults(3, 9, 15),
			info: &boruta.ListInfo{
				TotalItems:     3,
				RemainingItems: 0,
			},
			name: "filter by DONE state, higher priority and reqid%3 == 0",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{0, 3, 6, 9, 12, 15},
				States:     []boruta.ReqState{boruta.DONE},
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s:      nil,
			p:      nil,
			result: getResults(6, 12),
			info: &boruta.ListInfo{
				TotalItems:     2,
				RemainingItems: 0,
			},
			name: "filter by DONE state, lower priority and reqid%3 == 0",
		},
		{
			f: &filter.Requests{
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      nil,
			p:      nil,
			result: getResults(1, 3, 5, 7, 9, 11, 13, 15),
			info: &boruta.ListInfo{
				TotalItems:     8,
				RemainingItems: 0,
			},
			name: "filter by higher priority only",
		},
		{
			f: &filter.Requests{
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s:      nil,
			p:      nil,
			result: getResults(2, 4, 6, 8, 10, 12, 14, 16),
			info: &boruta.ListInfo{
				TotalItems:     8,
				RemainingItems: 0,
			},
			name: "filter by lower priority only",
		},
		{
			f: &filter.Requests{
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionBackward,
				Limit:     3,
			},
			result: getResults(10, 8, 6),
			info: &boruta.ListInfo{
				TotalItems:     8,
				RemainingItems: 3,
			},
			name: "backward paginator with ID not belonging to results",
		},
		{
			f: &filter.Requests{
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(4),
				Direction: boruta.DirectionBackward,
				Limit:     3,
			},
			result: getResults(10, 8, 6),
			info: &boruta.ListInfo{
				TotalItems:     8,
				RemainingItems: 3,
			},
			name: "backward paginator with ID belonging to results",
		},
		{
			f: &filter.Requests{
				Priorities: []boruta.Priority{req.Priority + 1},
			},
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionForward,
				Limit:     3,
			},
			result: getResults(4, 2),
			info: &boruta.ListInfo{
				TotalItems:     8,
				RemainingItems: 0,
			},
			name: "forward paginator with ID not belonging to results",
		},
		{
			f: &filter.Requests{
				States: []boruta.ReqState{boruta.WAIT},
			},
			s:      si,
			p:      nil,
			result: getResults(2, 1),
			info: &boruta.ListInfo{
				TotalItems:     2,
				RemainingItems: 0,
			},
			name: "filter by WAIT state only",
		},
		{
			f: &filter.Requests{
				States: []boruta.ReqState{boruta.DONE},
			},
			s:      nil,
			p:      nil,
			result: getResults(3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     14,
				RemainingItems: 0,
			},
			name: "filter by DONE state only",
		},
		{
			f: &filter.Requests{
				IDs: []boruta.ReqID{0, 3, 6, 9, 12, 15},
			},
			s:      nil,
			p:      nil,
			result: getResults(3, 6, 9, 12, 15),
			info: &boruta.ListInfo{
				TotalItems:     5,
				RemainingItems: 0,
			},
			name: "filter by reqid%3 == 0 only",
		},
		{
			f:      nil,
			s:      si,
			p:      nil,
			result: getResults(16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "nil filter",
		},
		{
			f:      &filter.Requests{},
			s:      nil,
			p:      nil,
			result: getResults(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "empty filter",
		},
		{
			f:      nil,
			s:      nil,
			p:      nil,
			result: getResults(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "all nil",
		},
		{
			f: nil,
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionForward,
				Limit:     3,
			},
			result: getResults(4, 3, 2),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 1,
			},
			name: "get page after item with page size smaller than nr of items",
		},
		{
			f: nil,
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionBackward,
				Limit:     3,
			},
			result: getResults(8, 7, 6),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 8,
			},
			name: "get page before item with page size smaller than nr of items",
		},
		{
			f: nil,
			s: nil,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionBackward,
				Limit:     32,
			},
			result: getResults(1, 2, 3, 4),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "get page before item with page size bigger than nr of items",
		},
		{
			f: nil,
			s: nil,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionForward,
				Limit:     32,
			},
			result: getResults(6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "get page after item with page size bigger than nr of items",
		},
		{
			f: nil,
			s: nil,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(5),
				Direction: boruta.DirectionForward,
				Limit:     0,
			},
			result: getResults(6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "get page after item with default page size",
		},
		{
			f: nil,
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(1),
				Direction: boruta.DirectionForward,
				Limit:     32,
			},
			result: getResults(),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "get page after last item",
		},
		{
			f: nil,
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(16),
				Direction: boruta.DirectionBackward,
				Limit:     32,
			},
			result: getResults(),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 0,
			},
			name: "get page before 1st item",
		},
		{
			f: nil,
			s: nil,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(0),
				Direction: boruta.DirectionBackward,
				Limit:     5,
			},
			result: getResults(12, 13, 14, 15, 16),
			info: &boruta.ListInfo{
				TotalItems:     16,
				RemainingItems: 11,
			},
			name: "get first page when paginating backwards",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{notFoundReqID},
				States:     []boruta.ReqState{notFoundState},
				Priorities: []boruta.Priority{notFoundPrio},
			},
			s: si,
			p: nil,
			info: &boruta.ListInfo{
				TotalItems:     0,
				RemainingItems: 0,
			},
			result: getResults(),
			name:   "missing state, priority and reqid in filter",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{0, 3, 6, 9, 12, 15},
				States:     []boruta.ReqState{boruta.WAIT},
				Priorities: []boruta.Priority{notFoundPrio},
			},
			s:      si,
			p:      nil,
			result: getResults(),
			info: &boruta.ListInfo{
				TotalItems:     0,
				RemainingItems: 0,
			},
			name: "missing priority in filter",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{0, 3, 6, 9, 12, 15},
				States:     []boruta.ReqState{notFoundState},
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      si,
			p:      nil,
			result: getResults(),
			info: &boruta.ListInfo{
				TotalItems:     0,
				RemainingItems: 0,
			},
			name: "missing state in filter",
		},
		{
			f: &filter.Requests{
				IDs:        []boruta.ReqID{notFoundReqID},
				States:     []boruta.ReqState{boruta.WAIT},
				Priorities: []boruta.Priority{req.Priority},
			},
			s:      si,
			p:      nil,
			result: getResults(),
			info: &boruta.ListInfo{
				TotalItems:     0,
				RemainingItems: 0,
			},
			name: "missing reqid in filter",
		},
		{
			f:    nil,
			s:    &boruta.SortInfo{Item: "foobarbaz"},
			p:    nil,
			info: nil,
			err:  boruta.ErrWrongSortItem,
			name: "wrong sort item",
		},
		{
			f: nil,
			s: si,
			p: &boruta.RequestsPaginator{
				ID:        boruta.ReqID(32),
				Direction: boruta.DirectionForward,
				Limit:     0,
			},
			info: nil,
			err:  boruta.NotFoundError("request"),
			name: "get page after item with unknown ID page size",
		},
	}

	checkReqs := func(assert *assert.Assertions, name string, reqs []*boruta.ReqInfo,
		resp []boruta.ReqInfo) {

		assert.Equal(len(reqs), len(resp), name)
		for i := range resp {
			assert.Equal(reqs[i], &resp[i], name)
		}
	}
	for _, test := range filterTests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)
			resp, info, err := rqueue.ListRequests(test.f, test.s, test.p)
			assert.Equal(test.err, err, test.name)
			assert.Equal(test.info, info, test.name)
			checkReqs(assert, test.name, test.result, resp)
		})
	}

	name := "nil interface"
	t.Run(name, func(t *testing.T) {
		assert := assert.New(t)
		resp, info, err := rqueue.ListRequests(nil, nil, nil)
		assert.Nil(err)
		assert.Equal(&boruta.ListInfo{
			TotalItems:     16,
			RemainingItems: 0,
		}, info)
		checkReqs(assert, name, getResults(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13,
			14, 15, 16), resp)
	})

	name = "big queue"
	// As tests are running in parallel rqueue modification would make some of previously defined
	// tests fail.
	ctrl2 := gomock.NewController(t)
	defer ctrl2.Finish()
	wm := NewMockWorkersManager(ctrl2)
	jm := NewMockJobsManager(ctrl2)
	wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""),
		nil).AnyTimes()
	wm.EXPECT().SetChangeListener(gomock.Any())
	rqueue2 := NewRequestQueue(wm, jm)
	for i := 0; i < boruta.MaxPageLimit+reqsCnt; i++ {
		rqueue2.NewRequest(req.Caps, req.Priority, req.Owner, tomorrow, nextWeek)
	}
	assert.Equal(t, boruta.MaxPageLimit+reqsCnt, len(rqueue2.requests))
	t.Run(name, func(t *testing.T) {
		assert := assert.New(t)
		resp, info, err := rqueue2.ListRequests(nil, nil, &boruta.RequestsPaginator{Limit: 0})
		assert.Nil(err)
		assert.EqualValues(boruta.MaxPageLimit+reqsCnt, info.TotalItems)
		assert.EqualValues(boruta.MaxPageLimit, len(resp))
		assert.EqualValues(reqsCnt, info.RemainingItems)
	})
}

func TestAcquireWorker(t *testing.T) {
	assert, rqueue, ctrl, jm := initTest(t)
	defer finiTest(rqueue, ctrl)
	req := requestsTests[0].req
	empty := boruta.AccessInfo{}
	testErr := errors.New("Test Error")

	// Add valid request.
	reqid, err := rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	assert.Nil(err)

	states := [...]boruta.ReqState{boruta.WAIT, boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT,
		boruta.DONE, boruta.FAILED, boruta.INPROGRESS}
	for _, state := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = state
		rqueue.mutex.Unlock()
		ainfo, err := rqueue.AcquireWorker(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
		assert.Equal(empty, ainfo)
	}

	// Try to acquire worker for non-existing request.
	ainfo, err := rqueue.AcquireWorker(boruta.ReqID(2))
	assert.Equal(boruta.NotFoundError("Request"), err)
	assert.Equal(empty, ainfo)

	// Try to acquire worker when jobs.Get() fails.
	jobInfo := boruta.JobInfo{
		WorkerUUID: "Test WorkerUUID",
	}
	rqueue.mutex.Lock()
	rqueue.requests[reqid].Job = &jobInfo
	rqueue.mutex.Unlock()
	ignoredJob := &workers.Job{Req: boruta.ReqID(0xBAD)}
	jm.EXPECT().Get(jobInfo.WorkerUUID).Return(ignoredJob, testErr)
	ainfo, err = rqueue.AcquireWorker(reqid)
	assert.Equal(testErr, err)
	assert.Equal(empty, ainfo)

	// AcquireWorker to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	job := &workers.Job{
		Access: boruta.AccessInfo{Addr: &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4)}},
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

	states := [...]boruta.ReqState{boruta.WAIT, boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT,
		boruta.DONE, boruta.FAILED, boruta.INPROGRESS}
	for _, state := range states {
		rqueue.mutex.Lock()
		rqueue.requests[reqid].State = state
		rqueue.mutex.Unlock()
		err = rqueue.ProlongAccess(reqid)
		assert.Equal(ErrWorkerNotAssigned, err)
	}

	// Try to prolong access of job for non-existing request.
	err = rqueue.ProlongAccess(boruta.ReqID(2))
	assert.Equal(boruta.NotFoundError("Request"), err)

	// ProlongAccess to succeed needs JobInfo to be set. It also needs to be
	// in INPROGRESS state, which was set in the loop.
	rqueue.mutex.Lock()
	rqueue.requests[reqid].Job = new(boruta.JobInfo)
	rqueue.mutex.Unlock()
	err = rqueue.ProlongAccess(reqid)
	assert.Nil(err)
}
