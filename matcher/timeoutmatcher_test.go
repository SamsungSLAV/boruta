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
	. "github.com/SamsungSLAV/boruta"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimeoutMatcher", func() {
	var ctrl *gomock.Controller
	var r *MockRequestsManager
	var m Matcher

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		r = NewMockRequestsManager(ctrl)
		m = NewTimeoutMatcher(r)
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	Describe("NewTimeoutMatcher", func() {
		It("should create a new TimeoutMatcher", func() {
			Expect(m).NotTo(BeNil())
		})
	})
	Describe("Notify", func() {
		It("should not call any methods on empty requests", func() {
			reqs := make([]ReqID, 0)

			m.Notify(reqs)
		})
		It("should run Close for each reqID passed to Notify and ignore returned errors", func() {
			err := NotFoundError("Request")
			reqs := []ReqID{1, 2, 5, 7}

			gomock.InOrder(
				r.EXPECT().Close(reqs[0]).Return(err),
				r.EXPECT().Close(reqs[1]).Return(nil),
				r.EXPECT().Close(reqs[2]).Return(nil),
				r.EXPECT().Close(reqs[3]).Return(err),
			)

			m.Notify(reqs)
		})
	})
})
