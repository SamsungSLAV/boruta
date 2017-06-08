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

package workers_test

import (
	. "git.tizen.org/tools/boruta"
	. "git.tizen.org/tools/boruta/workers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WorkerList", func() {
	var wl *WorkerList
	BeforeEach(func() {
		wl = NewWorkerList()
	})

	It("should return ErrNotImplemented", func() {
		var (
			err    error
			uuid   WorkerUUID   = ""
			caps   Capabilities = nil
			groups Groups       = nil
		)

		By("SetState")
		err = wl.SetState(uuid, MAINTENANCE)
		Expect(err).To(Equal(ErrNotImplemented))

		By("SetGroups")
		err = wl.SetGroups(uuid, groups)
		Expect(err).To(Equal(ErrNotImplemented))

		By("Deregister")
		err = wl.Deregister(uuid)
		Expect(err).To(Equal(ErrNotImplemented))

		By("ListWorkers")
		_, err = wl.ListWorkers(groups, caps)
		Expect(err).To(Equal(ErrNotImplemented))

		By("GetWorkerInfo")
		_, err = wl.GetWorkerInfo(uuid)
		Expect(err).To(Equal(ErrNotImplemented))
	})
})
