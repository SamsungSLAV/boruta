/*
 *  Copyright (c) 2018-2022 Samsung Electronics Co., Ltd All Rights Reserved
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
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/workers"
)

var (
	noWorker         = boruta.WorkerUUID("")
	testCapabilities = boruta.Capabilities{"key": "value"}
	testPriority     = (boruta.HiPrio + boruta.LoPrio) / 2
	testUser         = boruta.UserInfo{Groups: []boruta.Group{"Test Group"}}
	trigger          = make(chan int, 1)
)

func setTrigger(val int) { trigger <- val }

func eventuallyTrigger(val int) func() bool {
	return func() bool {
		select {
		case v := <-trigger:
			return val == v
		default:
			return false
		}
	}
}

func eventuallyState(s *RequestsTestSuite, reqid boruta.ReqID, state boruta.ReqState) func() bool {
	return func() bool {
		s.T().Helper()
		info, err := s.rqueue.GetRequestInfo(reqid)
		s.NoError(err)
		s.NotZero(reqid)
		return info.State == state
	}
}

// ValidMatcher should do nothing if there are no waiting requests.
func (s *RequestsTestSuite) TestOnWorkerIdleNoRequests() {
	testWorker := boruta.WorkerUUID("Test Worker UUID")
	s.rqueue.OnWorkerIdle(testWorker)
}

func (s *RequestsTestSuite) TestOnWorkerIdleMatchRequest() {
	testWorker := boruta.WorkerUUID("Test Worker UUID")
	// Add Request. Use trigger to wait for ValidMatcher goroutine to match worker.
	s.wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).DoAndReturn(func(boruta.Groups, boruta.Capabilities) (boruta.WorkerUUID, error) {
		setTrigger(1)
		return noWorker, s.testErr
	})
	reqid, err := s.rqueue.NewRequest(testCapabilities, testPriority, testUser, now, tomorrow)
	s.NoError(err)
	s.NotZero(reqid)
	s.Eventually(eventuallyTrigger(1), time.Second, 10*time.Millisecond)

	// Test. Use trigger to wait for ValidMatcher goroutine to match worker.
	s.wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).DoAndReturn(func(boruta.Groups, boruta.Capabilities) (boruta.WorkerUUID, error) {
		setTrigger(2)
		return noWorker, s.testErr
	})
	s.rqueue.OnWorkerIdle(testWorker)
	s.Eventually(eventuallyTrigger(2), time.Second, 10*time.Millisecond)

	// Updating request should also try to match worker.
	s.wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).DoAndReturn(func(boruta.Groups, boruta.Capabilities) (boruta.WorkerUUID, error) {
		setTrigger(3)
		return noWorker, s.testErr
	})
	rinfo, err := s.rqueue.GetRequestInfo(reqid)
	s.NoError(err)
	s.NotEmpty(rinfo)
	rinfo.Priority = rinfo.Priority + 1
	err = s.rqueue.UpdateRequest(&rinfo)
	s.NoError(err)
	s.Eventually(eventuallyTrigger(3), time.Second, 10*time.Millisecond)
}

func (s *RequestsTestSuite) TestOnWorkerFailed() {
	testWorker := boruta.WorkerUUID("Test Worker UUID")
	// When jobs.Get() fails OnWorkerFail should just return (without panic nor calling jobs.Finish()).
	s.jm.EXPECT().Get(testWorker).Return(nil, s.testErr)
	s.NotPanics(func() { s.rqueue.OnWorkerFail(testWorker) })

	// OnWorkerFail should panick when jobs.Get() returns unknown request ID.
	noReq := boruta.ReqID(0)
	job := workers.Job{Req: noReq}
	s.jm.EXPECT().Get(testWorker).Return(&job, nil)
	s.Panics(func() { s.rqueue.OnWorkerFail(testWorker) })

	// When call succeeds request should be in failed state.
	s.wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).DoAndReturn(func(boruta.Groups, boruta.Capabilities) (boruta.WorkerUUID, error) {
		setTrigger(4)
		return noWorker, s.testErr
	})
	reqid, err := s.rqueue.NewRequest(testCapabilities, testPriority, testUser, now, tomorrow)
	s.NoError(err)
	s.NotZero(reqid)
	// Wait until match is done.
	s.Eventually(eventuallyTrigger(4), time.Second, 10*time.Millisecond)
	job.Req = reqid
	s.jm.EXPECT().Get(testWorker).Return(&job, nil)
	s.jm.EXPECT().Finish(testWorker, false)
	s.NotPanics(func() { s.rqueue.OnWorkerFail(testWorker) })
	s.Eventually(eventuallyState(s, reqid, boruta.FAILED), time.Second, 10*time.Millisecond)
}
