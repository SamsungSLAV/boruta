/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
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
	"time"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Requests as WorkerChange", func() {
	var ctrl *gomock.Controller
	var wm *MockWorkersManager
	var jm *MockJobsManager
	var R *ReqsCollection
	testErr := errors.New("Test Error")
	testWorker := boruta.WorkerUUID("Test Worker UUID")
	noWorker := boruta.WorkerUUID("")
	testCapabilities := boruta.Capabilities{"key": "value"}
	testPriority := (boruta.HiPrio + boruta.LoPrio) / 2
	testUser := boruta.UserInfo{Groups: []boruta.Group{"Test Group"}}
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	trigger := make(chan int, 1)

	setTrigger := func(val int) {
		trigger <- val
	}
	eventuallyTrigger := func(val int) {
		EventuallyWithOffset(1, trigger).Should(Receive(Equal(val)))
	}
	eventuallyState := func(reqid boruta.ReqID, state boruta.ReqState) {
		EventuallyWithOffset(1, func() boruta.ReqState {
			info, err := R.GetRequestInfo(reqid)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, info).NotTo(BeNil())
			return info.State
		}).Should(Equal(state))
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		wm = NewMockWorkersManager(ctrl)
		jm = NewMockJobsManager(ctrl)
		wm.EXPECT().SetChangeListener(gomock.Any())
		R = NewRequestQueue(wm, jm)
	})
	AfterEach(func() {
		R.Finish()
		ctrl.Finish()
	})

	Describe("OnWorkerIdle", func() {
		It("ValidMatcher should do nothing if there are no waiting requests", func() {
			R.OnWorkerIdle(testWorker)
		})
		It("ValidMatcher should try matching request", func() {
			// Add Request. Use trigger to wait for ValidMatcher goroutine to match worker.
			wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).Return(noWorker, testErr).Do(func(boruta.Groups, boruta.Capabilities) {
				setTrigger(1)
			})
			reqid, err := R.NewRequest(testCapabilities, testPriority, testUser, now, tomorrow)
			Expect(err).NotTo(HaveOccurred())
			Expect(reqid).NotTo(BeZero())
			eventuallyTrigger(1)

			// Test. Use trigger to wait for ValidMatcher goroutine to match worker.
			wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).Return(noWorker, testErr).Do(func(boruta.Groups, boruta.Capabilities) {
				setTrigger(2)
			})
			R.OnWorkerIdle(testWorker)
			eventuallyTrigger(2)
		})
	})
	Describe("OnWorkerFail", func() {
		It("should return if jobs.Get fails", func() {
			jm.EXPECT().Get(testWorker).Return(nil, testErr)
			R.OnWorkerFail(testWorker)
		})
		It("should panic if failing worker was processing unknown Job", func() {
			noReq := boruta.ReqID(0)
			job := workers.Job{Req: noReq}
			jm.EXPECT().Get(testWorker).Return(&job, nil)
			Expect(func() {
				R.OnWorkerFail(testWorker)
			}).To(Panic())
		})
		It("should set request to FAILED state if call succeeds", func() {
			// Add Request. Use trigger to wait for ValidMatcher goroutine to match worker.
			wm.EXPECT().TakeBestMatchingWorker(testUser.Groups, testCapabilities).Return(noWorker, testErr).Do(func(boruta.Groups, boruta.Capabilities) {
				setTrigger(3)
			})
			reqid, err := R.NewRequest(testCapabilities, testPriority, testUser, now, tomorrow)
			Expect(err).NotTo(HaveOccurred())
			Expect(reqid).NotTo(BeZero())
			eventuallyTrigger(3)

			// Test.
			job := workers.Job{Req: reqid}
			jm.EXPECT().Get(testWorker).Return(&job, nil)
			jm.EXPECT().Finish(testWorker, false)
			R.OnWorkerFail(testWorker)
			eventuallyState(reqid, boruta.FAILED)
		})
	})
})
