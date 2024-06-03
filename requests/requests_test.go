/*
 *  Copyright (c) 2017-2022 Samsung Electronics Co., Ltd All Rights Reserved
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
	"net"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	"github.com/SamsungSLAV/boruta/workers"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	owner boruta.UserInfo
	job   boruta.JobInfo
	caps  = make(boruta.Capabilities)
)

var requestsTests = [...]struct {
	name string
	req  boruta.ReqInfo
	err  error
}{
	{
		name: "ValidRequest",
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
		name: "InvalidPriority",
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
		name: "ValidAfterNewerThenDeadline",
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
		name: "DeadlineInThePast",
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

func (s *RequestsTestSuite) TestNewRequestQueue() {
	s.rqueue.mutex.RLock()
	defer s.rqueue.mutex.RUnlock()
	s.Zero(len(s.rqueue.requests))
	s.NotNil(s.rqueue.queue)
	s.Zero(s.rqueue.queue.length)
}

func (s *RequestsTestSuite) TestNewRequest() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()

	s.Run("TimesAndPriorities", func() {
		for _, test := range requestsTests {
			test := test
			s.Run(test.name, func() {
				s.T().Parallel()
				reqid, err := s.rqueue.NewRequest(test.req.Caps, test.req.Priority,
					test.req.Owner, test.req.ValidAfter, test.req.Deadline)
				s.Equalf(test.req.ID, reqid, "test case: %s", test.name)
				s.Equalf(test.err, err, "test case: %s", test.name)
			})
		}

	})
	s.Run("ZeroTimes", func() {
		req := requestsTests[0].req
		start := time.Now()
		reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner,
			zeroTime, zeroTime)
		stop := time.Now()
		s.Nil(err)
		s.NotZero(reqid)
		s.rqueue.mutex.RLock()
		defer s.rqueue.mutex.RUnlock()
		res := s.rqueue.requests[reqid]
		s.True(start.Before(res.ValidAfter) && stop.After(res.ValidAfter))
		start = start.AddDate(0, 1, 0)
		stop = stop.AddDate(0, 1, 0)
		s.True(start.Before(res.Deadline) && stop.After(res.Deadline))
		s.EqualValues(2, s.rqueue.queue.length)
	})
}

func (s *RequestsTestSuite) TestCloseRequest() {
	req := requestsTests[0].req
	jobInfo := boruta.JobInfo{
		WorkerUUID: "Test WorkerUUID",
	}
	const reqsCnt = 4
	var reqs [reqsCnt]boruta.ReqID

	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	// Add valid request to the queue.
	for i := 0; i < reqsCnt; i++ {
		reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
		s.Nil(err)
		s.NotZero(reqid)
		reqs[i] = reqid
	}
	s.rqueue.queue.mtx.Lock()
	s.EqualValues(reqsCnt, s.rqueue.queue.length)
	s.rqueue.queue.mtx.Unlock()

	s.Run("Requests", func() {
		noreq := reqs[reqsCnt-1] + 1
		testCases := [...]struct {
			name  string
			reqid boruta.ReqID
			state boruta.ReqState
			sub   uint
			err   error
		}{
			{name: "MissingRequest", reqid: noreq, state: "", err: boruta.NotFoundError("Request")},
			{name: "Cancel", reqid: reqs[0], state: boruta.CANCEL, sub: 1, err: nil},
			{name: "Done", reqid: reqs[1], state: boruta.DONE, sub: 0, err: nil},  // Request is in progress and job is assigned.
			{name: "NoJob", reqid: reqs[2], state: boruta.DONE, sub: 0, err: nil}, // Request is in progress but job is missing.
		}

		s.jm.EXPECT().Finish(jobInfo.WorkerUUID, true)
		s.rqueue.mutex.Lock()
		s.rqueue.requests[reqs[1]].State = boruta.INPROGRESS
		s.rqueue.requests[reqs[1]].Job = &jobInfo
		s.rqueue.requests[reqs[2]].State = boruta.INPROGRESS
		s.rqueue.requests[reqs[2]].Job = nil
		s.rqueue.mutex.Unlock()
		for i := 1; i <= 2; i++ { // Requests that are in progress are removed from priority queue.
			reqinfo, err := s.rqueue.GetRequestInfo(reqs[i])
			s.NoError(err)
			s.rqueue.queue.removeRequest(&reqinfo)
		}

		for _, test := range testCases {
			test := test
			s.Run(test.name, func() {
				s.T().Parallel()
				if test.state == boruta.CANCEL {
					time.Sleep(time.Second)
				}
				s.rqueue.queue.mtx.Lock()
				qlen := s.rqueue.queue.length
				s.rqueue.queue.mtx.Unlock()
				// Adjust number of elements in the priority queue after closing.
				qlen -= test.sub
				s.ErrorIsf(test.err, s.rqueue.CloseRequest(test.reqid), "test case: %s", test.name)
				s.rqueue.queue.mtx.Lock()
				s.Equalf(qlen, s.rqueue.queue.length, "test case: %s", test.name)
				s.rqueue.queue.mtx.Unlock()
				s.rqueue.mutex.RLock()
				s.EqualValuesf(reqsCnt, len(s.rqueue.requests), "test case: %s", test.name)
				s.rqueue.mutex.RUnlock()
				if test.err == nil {
					s.rqueue.mutex.RLock()
					s.Equalf(test.state, s.rqueue.requests[test.reqid].State, "test case: %s", test.name)
					s.rqueue.mutex.RUnlock()
				}
			})
		}
	})

	// Simulation for the rest of states.
	s.Run("States", func() {
		states := [...]boruta.ReqState{boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT, boruta.DONE, boruta.FAILED}
		reqid := reqs[len(reqs)-1]
		reqinfo, err := s.rqueue.GetRequestInfo(reqid)
		s.Nil(err)
		s.rqueue.queue.removeRequest(&reqinfo)
		for _, state := range states {
			state := state
			s.rqueue.mutex.Lock()
			s.rqueue.requests[reqid].State = state
			s.rqueue.mutex.Unlock()
			s.Run(string(state), func() {
				s.T().Parallel()
				s.EqualValuesf(ErrModificationForbidden, s.rqueue.CloseRequest(reqid), "state: %s", state)
			})
		}
	})
	s.rqueue.mutex.RLock()
	// rqueue shouldn't change
	s.EqualValues(reqsCnt, len(s.rqueue.requests))
	s.rqueue.mutex.RUnlock()
	s.rqueue.queue.mtx.Lock()
	// Priority queue should be empty.
	s.EqualValues(0, s.rqueue.queue.length)
	s.rqueue.queue.mtx.Unlock()
}

func (s *RequestsTestSuite) TestUpdateRequest() {
	tmp := requestsTests[0].req

	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	// Add valid request.
	reqid, err := s.rqueue.NewRequest(tmp.Caps, tmp.Priority, tmp.Owner, tmp.ValidAfter, tmp.Deadline)
	s.NoError(err)
	s.NotZero(reqid)
	s.rqueue.mutex.RLock()
	req := s.rqueue.requests[reqid]
	s.rqueue.mutex.RUnlock()

	s.Run("Requests", func() {
		s.rqueue.mutex.RLock()
		reqUpdate := *req
		s.rqueue.mutex.RUnlock()
		reqBefore, err := s.rqueue.GetRequestInfo(reqid)
		s.NoError(err)
		// Check noop.
		s.NoError(s.rqueue.UpdateRequest(nil))
		reqUpdate.ValidAfter = zeroTime
		reqUpdate.Deadline = zeroTime
		reqUpdate.Priority = boruta.Priority(0)
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(req, &reqBefore)
		// Check request that doesn't exist.
		reqUpdate = *req
		s.rqueue.mutex.RUnlock()
		reqUpdate.ID++
		s.ErrorIs(boruta.NotFoundError("Request"), s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		reqUpdate.ID = req.ID
		// Change Priority only.
		reqUpdate.Priority = req.Priority - 1
		s.rqueue.mutex.RUnlock()
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(reqUpdate.Priority, req.Priority)
		s.rqueue.mutex.RUnlock()
		// Change ValidAfter only.
		reqUpdate.ValidAfter = yesterday
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(reqUpdate.ValidAfter, req.ValidAfter)
		s.rqueue.mutex.RUnlock()
		// Change Deadline only.
		reqUpdate.Deadline = tomorrow.AddDate(0, 0, 1).UTC()
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(reqUpdate.Deadline, req.Deadline)
		s.rqueue.mutex.RUnlock()
		// Change Priority, ValidAfter and Deadline.
		reqUpdate.Deadline = tomorrow
		reqUpdate.ValidAfter = time.Now().Add(time.Hour)
		reqUpdate.Priority = boruta.LoPrio
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(reqUpdate, *req)
		s.rqueue.mutex.RUnlock()
		// Change values to the same ones that are already set.
		s.NoError(s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		s.Equal(reqUpdate, *req)
		s.rqueue.mutex.RUnlock()
		// Change Priority to illegal value.
		reqUpdate.Priority = boruta.LoPrio + 1
		s.ErrorIs(ErrPriority, s.rqueue.UpdateRequest(&reqUpdate))
		s.rqueue.mutex.RLock()
		reqUpdate.Priority = req.Priority
		s.rqueue.mutex.RUnlock()
		//Change Deadline to illegal value.
		reqUpdate.Deadline = yesterday
		s.ErrorIs(ErrDeadlineInThePast, s.rqueue.UpdateRequest(&reqUpdate))
		reqUpdate.Deadline = time.Now().Add(time.Minute)
		s.ErrorIs(ErrInvalidTimeRange, s.rqueue.UpdateRequest(&reqUpdate))
		// Change ValidAfer to illegal value.
		s.rqueue.mutex.RLock()
		reqUpdate.ValidAfter = req.Deadline.Add(time.Hour)
		s.rqueue.mutex.RUnlock()
		s.ErrorIs(ErrInvalidTimeRange, s.rqueue.UpdateRequest(&reqUpdate))
	})
	// Try to change values for other changes.
	s.Run("States", func() {
		states := [...]boruta.ReqState{boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT, boruta.DONE, boruta.FAILED, boruta.INPROGRESS}
		reqUpdate := *req
		for _, state := range states {
			state := state
			s.Run(string(state), func() {
				s.T().Parallel()
				s.rqueue.mutex.Lock()
				s.rqueue.requests[reqid].State = state
				s.rqueue.mutex.Unlock()
				s.ErrorIsf(ErrModificationForbidden, s.rqueue.UpdateRequest(&reqUpdate), "state: %s", state)
			})
		}
	})
}

func (s *RequestsTestSuite) TestGetRequestInfo() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	req := requestsTests[0].req
	req.Job = nil
	reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	s.Nil(err)
	s.NotZero(reqid)
	noreq := reqid + 1

	testCases := [...]struct {
		name string
		id   boruta.ReqID
		req  boruta.ReqInfo
		err  error
	}{
		{name: "MissingRequest", id: noreq, req: boruta.ReqInfo{}, err: boruta.NotFoundError("Request")},
		{name: "ValidRequest", id: reqid, req: req, err: nil},
	}

	for _, test := range testCases {
		s.Run(test.name, func() {
			s.T().Parallel()
			r, err := s.rqueue.GetRequestInfo(test.id)
			s.Equalf(test.req, r, "test case: %s", test.name)
			s.ErrorIsf(test.err, err, "test case: %s", test.name)
		})
	}
}

func (s *RequestsTestSuite) TestListRequests() {
	req := requestsTests[0].req
	const reqsCnt = 16
	si := &boruta.SortInfo{
		Item:  "id",
		Order: boruta.SortOrderDesc,
	}

	getResults := func(ids ...int) (res []*boruta.ReqInfo) {
		res = make([]*boruta.ReqInfo, len(ids))
		for idx, id := range ids {
			res[idx] = s.rqueue.requests[boruta.ReqID(id)]
		}
		return
	}

	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	// Add few requests.
	for i := 0; i < reqsCnt; i++ {
		reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter,
			req.Deadline)
		if err != nil {
			s.T().Fatal("unable to create new request:", err)
		}
		if i%2 == 1 {
			s.rqueue.mutex.Lock()
			s.rqueue.requests[reqid].Priority++
			s.rqueue.mutex.Unlock()
		}
		if i > 1 {
			s.rqueue.mutex.Lock()
			s.rqueue.requests[reqid].State = boruta.DONE
			s.rqueue.mutex.Unlock()
		}
	}

	notFoundPrio := req.Priority - 1
	notFoundState := boruta.INVALID
	notFoundReqID := boruta.ReqID(reqsCnt + 1)
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

		s.T().Helper()
		assert.Equal(len(reqs), len(resp), name)
		for i := range resp {
			assert.Equalf(reqs[i], &resp[i], "test case: %s", name)
		}
	}
	for _, test := range filterTests {
		test := test
		s.Run(test.name, func() {
			s.T().Parallel()
			resp, info, err := s.rqueue.ListRequests(test.f, test.s, test.p)
			s.Equalf(test.err, err, "test case: %s", test.name)
			s.Equalf(test.info, info, "test case: %s", test.name)
			checkReqs(s.Assert(), test.name, test.result, resp)
		})
	}

	name := "nil interface"
	s.Run(name, func() {
		resp, info, err := s.rqueue.ListRequests(nil, nil, nil)
		s.Nil(err)
		s.Equal(&boruta.ListInfo{
			TotalItems:     16,
			RemainingItems: 0,
		}, info)
		checkReqs(s.Assert(), name, getResults(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13,
			14, 15, 16), resp)
	})

	name = "big queue"
	// As tests are running in parallel rqueue modification would make some of previously defined
	// tests fail.
	ctrl2 := gomock.NewController(s.T())
	defer ctrl2.Finish()
	wm := NewMockWorkersManager(ctrl2)
	jm := NewMockJobsManager(ctrl2)
	wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""),
		nil).AnyTimes()
	wm.EXPECT().SetChangeListener(gomock.Any())
	rqueue2 := NewRequestQueue(wm, jm)
	for i := 0; i < int(boruta.MaxPageLimit)+reqsCnt; i++ {
		_, err := rqueue2.NewRequest(req.Caps, req.Priority, req.Owner, tomorrow, nextWeek)
		s.Nil(err)
	}
	s.Equal(int(boruta.MaxPageLimit)+reqsCnt, len(rqueue2.requests))
	s.Run(name, func() {
		resp, info, err := rqueue2.ListRequests(nil, nil, &boruta.RequestsPaginator{Limit: 0})
		s.Nil(err)
		s.EqualValues(int(boruta.MaxPageLimit)+reqsCnt, info.TotalItems)
		s.EqualValues(boruta.MaxPageLimit, len(resp))
		s.EqualValues(reqsCnt, info.RemainingItems)
	})
}

func (s *RequestsTestSuite) TestAcquireWorker() {
	req := requestsTests[0].req
	empty := boruta.AccessInfo{}

	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	// Add valid requests.
	reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	s.Nil(err)
	s.NotZero(reqid)
	reqid2, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	s.Nil(err)
	s.NotZero(reqid2)
	noreq := reqid2 + 1

	s.Run("States", func() {
		states := [...]boruta.ReqState{boruta.WAIT, boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT,
			boruta.DONE, boruta.FAILED, boruta.INPROGRESS}
		for _, state := range states {
			state := state
			s.Run(string(state), func() {
				s.T().Parallel()
				s.rqueue.mutex.Lock()
				s.rqueue.requests[reqid].State = state
				s.rqueue.mutex.Unlock()
				ainfo, err := s.rqueue.AcquireWorker(reqid)
				s.Equalf(ErrWorkerNotAssigned, err, "state: %s", state)
				s.Equalf(empty, ainfo, "state: %s", state)
			})
		}
	})

	s.Run("Requests", func() {
		testCases := [...]struct {
			name    string
			id      boruta.ReqID
			job     *boruta.JobInfo
			mockJob *workers.Job
			err     error
		}{
			{name: "MissingRequest", id: noreq, job: nil, mockJob: nil, err: boruta.NotFoundError("Request")},
			{
				name:    "JobError",
				id:      reqid,
				job:     &boruta.JobInfo{WorkerUUID: "Worker1"},
				mockJob: &workers.Job{Req: boruta.ReqID(0xBAD)},
				err:     s.testErr,
			},
			{
				name:    "ValidRequest",
				id:      reqid2,
				job:     &boruta.JobInfo{WorkerUUID: "Worker2"},
				mockJob: &workers.Job{Access: boruta.AccessInfo{Addr: &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4)}}},
				err:     nil,
			},
		}

		for _, test := range testCases {
			test := test
			if test.job != nil {
				s.rqueue.mutex.Lock()
				s.rqueue.requests[test.id].Job = test.job
				s.rqueue.requests[test.id].State = boruta.INPROGRESS
				s.rqueue.mutex.Unlock()
				s.jm.EXPECT().Get(test.job.WorkerUUID).Return(test.mockJob, test.err)
			}
			s.Run(test.name, func() {
				s.T().Parallel()
				_, err := s.rqueue.AcquireWorker(test.id)
				s.Equalf(test.err, err, "test case: %s", test.name)
			})
		}
	})
}

func (s *RequestsTestSuite) TestProlongAccess() {
	req := requestsTests[0].req

	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	// Add valid request.
	reqid, err := s.rqueue.NewRequest(req.Caps, req.Priority, req.Owner, req.ValidAfter, req.Deadline)
	s.Nil(err)
	s.NotZero(reqid)

	s.Run("States", func() {
		states := [...]boruta.ReqState{boruta.WAIT, boruta.INVALID, boruta.CANCEL, boruta.TIMEOUT,
			boruta.DONE, boruta.FAILED, boruta.INPROGRESS}
		for _, state := range states {
			state := state
			s.Run(string(state), func() {
				s.T().Parallel()
				s.rqueue.mutex.Lock()
				s.rqueue.requests[reqid].State = state
				s.rqueue.mutex.Unlock()
				s.Equalf(ErrWorkerNotAssigned, s.rqueue.ProlongAccess(reqid), "state: %s", state)
			})
		}
	})

	s.Run("Requests", func() {
		testCases := [...]struct {
			name string
			id   boruta.ReqID
			err  error
		}{
			{name: "Missing", id: reqid + 1, err: boruta.NotFoundError("Request")},
			{name: "Valid", id: reqid, err: nil},
		}
		// ProlongAccess to succeed needs JobInfo to be set. It also needs to be
		// in INPROGRESS state.
		s.rqueue.mutex.Lock()
		s.rqueue.requests[reqid].Job = new(boruta.JobInfo)
		s.rqueue.requests[reqid].State = boruta.INPROGRESS
		s.rqueue.mutex.Unlock()

		for _, test := range testCases {
			test := test
			s.Run(test.name, func() {
				s.T().Parallel()
				s.ErrorIsf(test.err, s.rqueue.ProlongAccess(test.id), "test case: %s", test.name)
			})
		}
	})
}
