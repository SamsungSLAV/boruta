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

//go:generate mockgen -package requests -destination=workersmanager_mock_test.go -write_package_comment=false github.com/SamsungSLAV/boruta/matcher WorkersManager
//go:generate mockgen -package requests -destination=jobsmanager_mock_test.go -write_package_comment=false github.com/SamsungSLAV/boruta/matcher JobsManager

import (
	"sync"
	"time"

	"github.com/SamsungSLAV/boruta"

	gomock "github.com/golang/mock/gomock"
)

func tryLock(mtx *sync.RWMutex, ch chan bool) {
	mtx.Lock()
	defer mtx.Unlock()
	ch <- true
}

func tryReceive(ch chan bool) func() bool {
	return func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}
}

func (s *RequestsTestSuite) TestInitIteration() {
	entered := make(chan bool)
	mutexUnlocked := tryReceive(entered)

	s.Run("ValidInit", func() {
		s.NoError(s.rqueue.InitIteration())
		s.True(s.rqueue.iterating)

		// Verify that mutex is locked.
		go tryLock(s.rqueue.mutex, entered)
		s.Never(mutexUnlocked, time.Second, 10*time.Millisecond)

		// Release the mutex
		s.rqueue.mutex.Unlock()
		s.Eventually(mutexUnlocked, time.Second, 10*time.Millisecond)
		s.TearDownTest()
	})

	s.Run("InitError", func() {
		s.SetupTest()
		s.rqueue.mutex.Lock()
		s.rqueue.iterating = true
		s.rqueue.mutex.Unlock()
		s.EqualError(s.rqueue.InitIteration(), boruta.ErrInternalLogicError.Error())
		go tryLock(s.rqueue.mutex, entered)
		s.Eventually(mutexUnlocked, time.Second, 10*time.Millisecond)
	})
}

func (s *RequestsTestSuite) TestTerminateIteration() {
	entered := make(chan bool)
	mutexUnlocked := tryReceive(entered)

	s.NoError(s.rqueue.InitIteration())
	s.True(s.rqueue.iterating)
	s.rqueue.TerminateIteration()
	// iterating is set to false and mutex is unlocked
	s.False(s.rqueue.iterating)
	go tryLock(s.rqueue.mutex, entered)
	s.Eventually(mutexUnlocked, time.Second, 10*time.Millisecond)

	// When iterating is false and mutex is locked, then TerminateIteration should unlock the mutex.
	s.rqueue.mutex.Lock()
	s.rqueue.TerminateIteration()
	s.False(s.rqueue.iterating) // iterating hasn't changed
	go tryLock(s.rqueue.mutex, entered)
	s.Eventually(mutexUnlocked, time.Second, 10*time.Millisecond)
}

func (s *RequestsTestSuite) TestIteration() {
	s.Panics(func() {
		s.rqueue.mutex.Lock()
		defer s.rqueue.mutex.Unlock()
		s.rqueue.Next()
	})

	verify := []boruta.ReqID{3, 5, 1, 2, 7, 4, 6}
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	insert := func(p boruta.Priority) {
		_, err := s.rqueue.NewRequest(boruta.Capabilities{}, p, boruta.UserInfo{}, now, tomorrow)
		s.NoError(err)
	}
	insert(3) //1
	insert(3) //2
	insert(1) //3
	insert(5) //4
	insert(1) //5
	insert(5) //6
	insert(3) //7

	s.Run("IterateOverAll", func() {
		reqs := make([]boruta.ReqID, 0, len(verify))

		s.NoError(s.rqueue.InitIteration())
		for r, ok := s.rqueue.Next(); ok; r, ok = s.rqueue.Next() {
			reqs = append(reqs, r)
		}
		s.rqueue.TerminateIteration()
		s.Equal(verify, reqs)
	})

	s.Run("RestartIterations", func() {
		for times := 0; times < len(verify); times++ {
			reqs := make([]boruta.ReqID, 0, len(verify))
			i := 0
			s.NoError(s.rqueue.InitIteration())
			for r, ok := s.rqueue.Next(); ok && i < times; r, ok = s.rqueue.Next() {
				reqs = append(reqs, r)
				i++
			}
			s.rqueue.TerminateIteration()
			s.Equal(verify[:times], reqs)
		}
	})

}

func (s *RequestsTestSuite) TestVerifyIfReady() {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	req, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
	s.NoError(err)
	s.NotZero(req)
	noreq := req + 1

	s.Run("Requests", func() {
		testCases := [...]struct {
			name string
			req  boruta.ReqID
			t    time.Time
			ret  bool
		}{
			{name: "UnknownReqID", req: noreq, t: now, ret: false},
			{name: "DeadlineOK", req: req, t: tomorrow.Add(-time.Hour), ret: true},
			{name: "OnDeadline", req: req, t: tomorrow, ret: false},
			{name: "DeadlinePassed", req: req, t: tomorrow.Add(time.Hour), ret: false},
			{name: "ValidAfterInFuture", req: req, t: now.Add(-time.Hour), ret: false},
			{name: "OnValidAfter", req: req, t: now, ret: true},
			{name: "ValidAfterPassed", req: req, t: now.Add(time.Hour), ret: true},
			// Request is known, in WAIT state and now is between ValidAfter and Deadline.
			{name: "ValidReqOnTime", req: req, t: now.Add(12 * time.Hour), ret: true},
		}

		for _, test := range testCases {
			s.Run(test.name, func() {
				s.T().Parallel()
				s.Equalf(test.ret, s.rqueue.VerifyIfReady(test.req, test.t), "test case: %s", test.name)
			})
		}
	})

	s.Run("States", func() {
		states := [...]boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID, boruta.DONE, boruta.FAILED}
		for _, state := range states {
			state := state
			s.rqueue.mutex.Lock()
			s.rqueue.requests[req].State = state
			s.rqueue.mutex.Unlock()
			s.Run(string(state), func() {
				s.T().Parallel()
				s.Falsef(s.rqueue.VerifyIfReady(req, now), "state: %s", state)
			})
		}
	})
}

func (s *RequestsTestSuite) TestGet() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()

	now := time.Now()
	req, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
	s.NoError(err)
	s.NotZero(req)
	noreq := req + 1

	testCases := [...]struct {
		name string
		id   boruta.ReqID
		err  error
	}{
		{name: "Missing", id: noreq, err: boruta.NotFoundError("Request")},
		{name: "Valid", id: req, err: nil},
	}

	for _, test := range testCases {
		test := test
		s.Run(test.name, func() {
			s.T().Parallel()
			r, err := s.rqueue.Get(test.id)
			s.ErrorIs(test.err, err)
			if err != nil {
				s.Empty(r)
			} else {
				s.rqueue.mutex.RLock()
				rinfo, ok := s.rqueue.requests[req]
				s.rqueue.mutex.RUnlock()
				s.Truef(ok, "test case: %s", test.name)
				s.Equalf(*rinfo, r, "test case: %s", test.name)
			}
		})
	}
}

func (s *RequestsTestSuite) TestTimeout() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	vafter := time.Now()

	s.Run("Requests", func() {
		future, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, vafter, tomorrow)
		s.NoError(err)
		s.NotZero(future)

		// As we want to trigger timeout manually for this request deadlineTimes is replaced with new one.
		// After adding request previous one is restored
		s.rqueue.mutex.Lock()
		deadlineTimes := s.rqueue.deadlineTimes
		s.rqueue.deadlineTimes = newRequestTimes()
		s.rqueue.mutex.Unlock()
		past, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, vafter, time.Now().Add(1*time.Second))
		s.NoError(err)
		s.NotZero(past)
		s.rqueue.mutex.Lock()
		// Restore previous deadlineTimes heap.
		s.rqueue.deadlineTimes = deadlineTimes
		s.rqueue.mutex.Unlock()

		noreq := past + 1

		testCases := [...]struct {
			name   string
			req    boruta.ReqID
			before uint
			after  uint
			sleep  time.Duration
			state  boruta.ReqState
			err    error
		}{
			{name: "MissingRequest", req: noreq, before: 2, after: 2, sleep: 0 * time.Second, state: "", err: boruta.NotFoundError("Request")},
			{name: "DeadlineInFuture", req: future, before: 2, after: 2, sleep: 0 * time.Second, state: boruta.WAIT, err: ErrModificationForbidden},
			{name: "DeadlinePassed", req: past, before: 2, after: 1, sleep: 1 * time.Second, state: boruta.TIMEOUT, err: nil},
		}

		for _, test := range testCases {
			test := test
			s.Run(test.name, func() {
				s.T().Parallel()
				s.rqueue.queue.mtx.Lock()
				s.Equalf(test.before, s.rqueue.queue.length, "test case: %s", test.name)
				s.rqueue.queue.mtx.Unlock()
				time.Sleep(test.sleep)
				s.ErrorIsf(test.err, s.rqueue.Timeout(test.req), "test case: %s", test.name)
				s.rqueue.queue.mtx.Lock()
				s.Equalf(test.after, s.rqueue.queue.length, "test case: %s", test.name)
				s.rqueue.queue.mtx.Unlock()
				s.rqueue.mutex.RLock()
				if test.state != "" {
					s.Equalf(test.state, s.rqueue.requests[test.req].State, "test case: %s", test.name)
				}
				s.rqueue.mutex.RUnlock()
			})
		}
	})

	s.Run("States", func() {
		// Timeout should only work on requests in WAIT state.
		states := [...]boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID, boruta.DONE, boruta.FAILED}
		req, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, vafter, tomorrow)
		s.NoError(err)
		s.NotZero(req)

		for _, state := range states {
			state := state
			s.rqueue.mutex.Lock()
			s.rqueue.requests[req].State = state
			s.rqueue.mutex.Unlock()
			s.rqueue.queue.mtx.Lock()
			qlen := s.rqueue.queue.length
			s.rqueue.queue.mtx.Unlock()
			s.Run(string(state), func() {
				s.T().Parallel()
				s.rqueue.queue.mtx.Lock()
				s.EqualValuesf(qlen, s.rqueue.queue.length, "state: %s", state)
				s.rqueue.queue.mtx.Unlock()
				s.ErrorIsf(ErrModificationForbidden, s.rqueue.Timeout(req), "state: %s", state)
				s.rqueue.queue.mtx.Lock()
				s.EqualValuesf(qlen, s.rqueue.queue.length, "state: %s", state)
				s.rqueue.queue.mtx.Unlock()
			})
		}
	})
}

func (s *RequestsTestSuite) TestClose() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()

	now := time.Now()
	const reqsCnt = 3
	var reqs [reqsCnt]boruta.ReqID
	for i := 0; i < reqsCnt; i++ {
		req, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
		s.NoError(err)
		s.NotZero(req)
		reqs[i] = req
	}

	s.Run("JobsAndRequests", func() {
		testWorker := boruta.WorkerUUID("TestWorker")
		noreq := reqs[reqsCnt-1] + 1
		testCases := [...]struct {
			name   string
			req    boruta.ReqID
			job    *boruta.JobInfo
			before boruta.ReqState
			after  boruta.ReqState
			err    error
		}{
			{name: "MissingRequest", req: noreq, job: nil, before: "", after: "", err: boruta.NotFoundError("Request")},
			{
				name:   "MissingJob",
				req:    reqs[0],
				job:    nil,
				before: boruta.INPROGRESS,
				after:  boruta.INPROGRESS,
				err:    boruta.ErrInternalLogicError,
			},
			{
				name:   "JobRunning",
				req:    reqs[0],
				job:    &boruta.JobInfo{Timeout: tomorrow},
				before: boruta.INPROGRESS,
				after:  boruta.INPROGRESS,
				err:    ErrModificationForbidden,
			},
			{
				name:   "CloseAndReleaseWorker",
				req:    reqs[1],
				job:    &boruta.JobInfo{Timeout: now.AddDate(0, 0, -1), WorkerUUID: testWorker},
				before: boruta.INPROGRESS,
				after:  boruta.DONE,
				err:    nil,
			},
		}

		// Successful Close will finish the job, so Finish will be called only once.
		s.jm.EXPECT().Finish(testWorker, true).Times(1)
		for _, test := range testCases {
			test := test
			s.Run(test.name, func() {
				s.T().Parallel()
				s.rqueue.mutex.Lock()
				rinfo, ok := s.rqueue.requests[test.req]
				if ok {
					rinfo.State = test.before
					rinfo.Job = test.job
				}
				s.rqueue.mutex.Unlock()
				s.ErrorIsf(test.err, s.rqueue.Close(test.req), "test case: %s", test.name)
				if test.after != "" {
					s.rqueue.mutex.RLock()
					s.Equalf(test.after, rinfo.State, "test case: %s", test.name)
					s.rqueue.mutex.RUnlock()
				}
			})
		}
	})

	// Only requests that are INPROGRESS can be closed.
	s.Run("RequestStates", func() {
		states := [...]boruta.ReqState{boruta.WAIT, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID, boruta.DONE, boruta.FAILED}
		for _, state := range states {
			state := state
			s.rqueue.mutex.Lock()
			s.rqueue.requests[reqs[2]].State = state
			s.rqueue.mutex.Unlock()
			s.Run(string(state), func() {
				s.T().Parallel()
				s.ErrorIsf(ErrModificationForbidden, s.rqueue.Close(reqs[2]), "state: %s", state)
			})
		}
	})
}

func (s *RequestsTestSuite) TestRun() {
	s.wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), s.testErr).AnyTimes()
	now := time.Now()
	req, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
	s.NoError(err)
	req2, err := s.rqueue.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
	s.NoError(err)
	testWorker := boruta.WorkerUUID("TestWorker")
	noreq := req2 + 1

	doChecks := func(s *RequestsTestSuite, r boruta.ReqID, worker boruta.WorkerUUID, after uint, iter bool, err error) {
		s.T().Helper()
		s.NoError(s.rqueue.InitIteration())
		s.EqualValues(2, s.rqueue.queue.length)
		s.True(s.rqueue.iterating)
		s.ErrorIs(err, s.rqueue.Run(r, worker))
		rinfo, ok := s.rqueue.requests[r]
		if ok && rinfo.Job != nil {
			s.Greater(rinfo.Job.Timeout, time.Now())
		}
		s.Equal(iter, s.rqueue.iterating)
		s.Equal(after, s.rqueue.queue.length)
		s.rqueue.TerminateIteration()
	}

	s.Run("States", func() {
		// Only requests that are WAIT can be run.
		states := [...]boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID, boruta.DONE, boruta.FAILED}
		for _, state := range states {
			state := state
			s.rqueue.mutex.Lock()
			s.rqueue.requests[req2].State = state
			s.rqueue.mutex.Unlock()
			s.Run(string(state), func() {
				s.T().Parallel()
				doChecks(s, req2, testWorker, 2, true, ErrModificationForbidden)
			})
		}
	})
	s.Run("UnknownID", func() {
		s.rqueue.mutex.Lock()
		defer s.rqueue.mutex.Unlock()
		s.rqueue.queue.mtx.Lock()
		s.EqualValues(2, s.rqueue.queue.length)
		s.rqueue.queue.mtx.Unlock()
		s.ErrorIs(boruta.NotFoundError("Request"), s.rqueue.Run(noreq, testWorker))
		s.rqueue.queue.mtx.Lock()
		s.EqualValues(2, s.rqueue.queue.length)
		s.rqueue.queue.mtx.Unlock()
	})
	s.Run("Iterating", func() {
		testCases := [...]struct {
			name   string
			req    boruta.ReqID
			worker boruta.WorkerUUID
			after  uint
			iter   bool
			err    error
		}{
			{name: "UnknownID", req: noreq, worker: testWorker, after: 2, iter: true, err: boruta.NotFoundError("Request")},
			{name: "ValidRequest", req: req, worker: testWorker, after: 1, iter: false, err: nil},
		}
		for _, test := range testCases {
			test := test
			s.Run(test.name, func() {
				doChecks(s, test.req, test.worker, test.after, test.iter, test.err)
			})
		}
	})
}
