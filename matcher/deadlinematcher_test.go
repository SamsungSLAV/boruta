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
	"github.com/SamsungSLAV/slav/logger"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeadlineMatcher", func() {
	var ctrl *gomock.Controller
	var r *MockRequestsManager
	var m Matcher

	BeforeEach(func() {
		logger.SetThreshold(logger.EmergLevel)
		ctrl = gomock.NewController(GinkgoT())
		r = NewMockRequestsManager(ctrl)
		m = NewDeadlineMatcher(r)
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	Describe("NewDeadlineMatcher", func() {
		It("should not use requests", func() {
			Expect(m).NotTo(BeNil())
		})
	})
	Describe("Notify", func() {
		It("should run no methods on empty requests", func() {
			reqs := make([]ReqID, 0)

			m.Notify(reqs)
		})
		It("should run Timeout for each reqID passed to Notify and ignore returned errors", func() {
			err := NotFoundError("Request")
			reqs := []ReqID{1, 2, 5, 7}

			gomock.InOrder(
				r.EXPECT().Timeout(reqs[0]).Return(err),
				r.EXPECT().Timeout(reqs[1]).Return(nil),
				r.EXPECT().Timeout(reqs[2]).Return(nil),
				r.EXPECT().Timeout(reqs[3]).Return(err),
			)

			m.Notify(reqs)
		})
	})
})
