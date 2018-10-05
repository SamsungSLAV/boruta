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

package matcher

import (
	"errors"
	"time"

	. "github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/workers"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValidMatcher", func() {
	var ctrl *gomock.Controller
	var r *MockRequestsManager
	var w *MockWorkersManager
	var j *MockJobsManager
	var m Matcher
	var pre time.Time

	zeroReq := ReqID(0)
	req := ReqID(101)
	groups := Groups{"A", "B", "C"}
	caps := Capabilities{"keyX": "valX", "keyY": "valY", "keyZ": "valZ"}
	worker := WorkerUUID("Test worker")
	reqInfo := ReqInfo{ID: req, Caps: caps, Owner: UserInfo{Groups: groups}}

	checkTime := func(_ ReqID, t time.Time) {
		Expect(t).To(BeTemporally(">=", pre))
		Expect(t).To(BeTemporally("<=", time.Now()))
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		r = NewMockRequestsManager(ctrl)
		w = NewMockWorkersManager(ctrl)
		j = NewMockJobsManager(ctrl)
		pre = time.Now()
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	Describe("NewValidMatcher", func() {
		It("should not use requests, workers nor jobs", func() {
			m := NewValidMatcher(r, w, j)
			Expect(m).NotTo(BeNil())
		})
	})
	Describe("Notify", func() {
		BeforeEach(func() {
			m = NewValidMatcher(r, w, j)
		})
		It("should not iterate over requests when InitIteration fails", func() {
			anyError := errors.New("test error")
			r.EXPECT().InitIteration().Return(anyError)

			Expect(func() {
				m.Notify(nil)
			}).To(Panic())
		})
		It("should run only Lock, Unlock, First on empty requests", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should ignore not-ready requests", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(false),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should continue iterating over requests when Get fails", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(true),
				r.EXPECT().Get(req).Return(ReqInfo{}, NotFoundError("Request")),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should match workers when Get succeeds", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(true),
				r.EXPECT().Get(req).Return(reqInfo, nil),
				w.EXPECT().TakeBestMatchingWorker(groups, caps).Return(WorkerUUID(""), workers.ErrWorkerNotFound),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should prepare worker without key generation when job creation fails", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(true),
				r.EXPECT().Get(req).Return(reqInfo, nil),
				w.EXPECT().TakeBestMatchingWorker(groups, caps).Return(worker, nil),
				j.EXPECT().Create(req, worker).Return(ErrJobAlreadyExists),
				w.EXPECT().PrepareWorker(worker, false),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should prepare worker without key generation when running request fails", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(true),
				r.EXPECT().Get(req).Return(reqInfo, nil),
				w.EXPECT().TakeBestMatchingWorker(groups, caps).Return(worker, nil),
				j.EXPECT().Create(req, worker),
				r.EXPECT().Run(req, worker).Return(NotFoundError("Request")),
				w.EXPECT().PrepareWorker(worker, false),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
		It("should create job when match is found and run request", func() {
			gomock.InOrder(
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(req, true),
				r.EXPECT().VerifyIfReady(req, gomock.Any()).Do(checkTime).Return(true),
				r.EXPECT().Get(req).Return(reqInfo, nil),
				w.EXPECT().TakeBestMatchingWorker(groups, caps).Return(worker, nil),
				j.EXPECT().Create(req, worker),
				r.EXPECT().Run(req, worker),
				r.EXPECT().TerminateIteration(),
				r.EXPECT().InitIteration(),
				r.EXPECT().Next().Return(zeroReq, false),
				r.EXPECT().TerminateIteration(),
			)

			m.Notify(nil)
		})
	})
})
