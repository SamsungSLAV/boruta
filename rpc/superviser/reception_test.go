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
	"errors"
	"net"
	"net/rpc"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("superviserReception", func() {
	var ctrl *gomock.Controller
	var msv *mocks.MockSuperviser

	const badAddr = "500.100.900"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		msv = mocks.NewMockSuperviser(ctrl)
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	It("should start superviserReception", func() {
		i, err := startSuperviserReception(msv, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(i).NotTo(BeNil())
		Expect(i.listener.Addr().Network()).To(Equal("tcp"))
		sshd, err := net.ResolveTCPAddr("tcp", i.listener.Addr().String())
		Expect(err).ToNot(HaveOccurred())
		Expect(sshd.IP.IsLoopback())
	})

	It("should fail to superviserReception if address is bad", func() {
		_, err := startSuperviserReception(msv, badAddr)
		Expect(err).To(HaveOccurred())
	})

	It("should pass StartSuperviserReception call to superviserReception and return error", func() {
		err := StartSuperviserReception(msv, badAddr)
		Expect(err).To(HaveOccurred())
	})

	Describe("with Superviser reception started", func() {
		var i *superviserReception
		var addr net.Addr
		uuidStr := "test-uuid"
		uuid := boruta.WorkerUUID(uuidStr)
		caps := boruta.Capabilities{"UUID": uuidStr}
		dryadRefAddr := net.TCPAddr{Port: 7175}
		sshRefAddr := net.TCPAddr{Port: 22}
		testErr := errors.New("test error")

		BeforeEach(func() {
			var err error
			i, err = startSuperviserReception(msv, "")
			Expect(err).ToNot(HaveOccurred())
			addr = i.listener.Addr()
		})

		It("should get IP from connection", func() {
			conn, err := net.Dial(addr.Network(), addr.String())
			Expect(err).ToNot(HaveOccurred())
			c := NewSuperviserClient(rpc.NewClient(conn))

			msv.EXPECT().Register(caps, gomock.Any(), gomock.Any()).DoAndReturn(
				func(c boruta.Capabilities, dryad string, ssh string) (err error) {
					dryadAddr, err0 := net.ResolveTCPAddr("tcp", dryad)
					Expect(err0).NotTo(HaveOccurred())
					Expect(dryadAddr.IP.IsLoopback()).To(BeTrue())
					Expect(dryadAddr.Port).To(Equal(dryadRefAddr.Port))
					sshAddr, err0 := net.ResolveTCPAddr("tcp", ssh)
					Expect(err0).NotTo(HaveOccurred())
					Expect(sshAddr.IP.IsLoopback()).To(BeTrue())
					Expect(sshAddr.Port).To(Equal(sshRefAddr.Port))
					return testErr
				})
			err = c.Register(caps, dryadRefAddr.String(), sshRefAddr.String())
			Expect(err).To(Equal(rpc.ServerError(testErr.Error())))
		})

		It("should get IP from argument", func() {
			c, err := DialSuperviserClient(addr.String())
			Expect(err).ToNot(HaveOccurred())

			msv.EXPECT().Register(caps, gomock.Any(), gomock.Any()).DoAndReturn(
				func(c boruta.Capabilities, dryad string, ssh string) (err error) {
					dryadAddr, err0 := net.ResolveTCPAddr("tcp", dryad)
					Expect(err0).NotTo(HaveOccurred())
					Expect(dryadAddr.IP.IsLoopback()).To(BeTrue())
					Expect(dryadAddr.Port).To(Equal(dryadRefAddr.Port))
					sshAddr, err0 := net.ResolveTCPAddr("tcp", ssh)
					Expect(err0).NotTo(HaveOccurred())
					Expect(sshAddr.IP.IsLoopback()).To(BeTrue())
					Expect(sshAddr.Port).To(Equal(sshRefAddr.Port))
					return testErr
				})
			ip := net.IPv4(127, 0, 0, 1)
			da := net.TCPAddr{IP: ip, Port: dryadRefAddr.Port}
			sa := net.TCPAddr{IP: ip, Port: sshRefAddr.Port}
			err = c.Register(caps, da.String(), sa.String())
			Expect(err).To(Equal(rpc.ServerError(testErr.Error())))
		})

		It("should fail to call with either address empty", func() {
			c, err := DialSuperviserClient(addr.String())
			Expect(err).ToNot(HaveOccurred())

			err = c.Register(boruta.Capabilities{"UUID": uuidStr}, "", sshRefAddr.String())
			Expect(err).To(HaveOccurred())

			err = c.Register(boruta.Capabilities{"UUID": uuidStr}, dryadRefAddr.String(), "")
			Expect(err).To(HaveOccurred())
		})

		It("should call superviser's SetFail method", func() {
			testReason := "test reason"

			c, err := DialSuperviserClient(addr.String())
			Expect(err).ToNot(HaveOccurred())

			msv.EXPECT().SetFail(uuid, testReason).Return(testErr)
			err = c.SetFail(uuid, testReason)
			Expect(err).To(Equal(rpc.ServerError(testErr.Error())))
		})

		It("should properly close connection", func() {
			c, err := DialSuperviserClient(addr.String())
			Expect(err).ToNot(HaveOccurred())

			c.Close()
		})
	})
})
