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

package superviser

import (
	"net"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("superviserReception", func() {
	var i *superviserReception
	var wl *workers.WorkerList
	var addr net.Addr

	BeforeEach(func() {
		var err error
		wl = workers.NewWorkerList()
		i, err = startSuperviserReception(wl, "")
		Expect(err).ToNot(HaveOccurred())
		addr = i.listener.Addr()
	})

	It("should get IP from connection", func() {
		uuidStr := "test-uuid"
		uuid := boruta.WorkerUUID(uuidStr)
		c, err := DialSuperviserClient(addr.String())
		Expect(err).ToNot(HaveOccurred())

		err = c.Register(boruta.Capabilities{"UUID": uuidStr})
		Expect(err).ToNot(HaveOccurred())

		ip, err := wl.GetWorkerIP(uuid)
		Expect(err).ToNot(HaveOccurred())
		Expect(ip).ToNot(BeNil())
	})
})
