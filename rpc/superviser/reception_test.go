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
	"net/rpc"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("superviserReception", func() {
	var i *superviserReception
	var wl *workers.WorkerList
	var addr net.Addr
	uuidStr := "test-uuid"
	refAddr := net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 7175,
	}

	BeforeEach(func() {
		var err error
		wl = workers.NewWorkerList()
		i, err = startSuperviserReception(wl, "")
		Expect(err).ToNot(HaveOccurred())
		addr = i.listener.Addr()
	})

	It("should get IP from connection", func() {
		uuid := boruta.WorkerUUID(uuidStr)
		conn, err := net.Dial(addr.Network(), addr.String())
		Expect(err).ToNot(HaveOccurred())
		c := NewSuperviserClient(rpc.NewClient(conn))

		err = c.Register(boruta.Capabilities{"UUID": uuidStr}, ":7175", ":22")
		Expect(err).ToNot(HaveOccurred())

		addr, err := wl.GetWorkerAddr(uuid)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr.Port).To(Equal(refAddr.Port))
		Expect(addr.IP.IsLoopback()).To(BeTrue())
	})

	It("should get IP from argument", func() {
		uuid := boruta.WorkerUUID(uuidStr)
		c, err := DialSuperviserClient(addr.String())
		Expect(err).ToNot(HaveOccurred())

		err = c.Register(boruta.Capabilities{"UUID": uuidStr}, refAddr.String(), refAddr.IP.String()+":22")
		Expect(err).ToNot(HaveOccurred())

		addr, err := wl.GetWorkerAddr(uuid)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr).To(Equal(refAddr))
	})

	It("should fail to call with either address empty", func() {
		c, err := DialSuperviserClient(addr.String())
		Expect(err).ToNot(HaveOccurred())

		err = c.Register(boruta.Capabilities{"UUID": uuidStr}, "", refAddr.IP.String()+":22")
		Expect(err).To(HaveOccurred())

		err = c.Register(boruta.Capabilities{"UUID": uuidStr}, refAddr.String(), "")
		Expect(err).To(HaveOccurred())
	})
})
