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
package tunnels

import (
	"io"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tunnels", func() {
	invalidIP := net.IPv4(255, 8, 8, 8)

	listen := func(done chan struct{}, in, out string) *net.TCPAddr {
		addr := new(net.TCPAddr)
		ln, err := net.ListenTCP("tcp", addr)
		go func() {
			defer close(done)
			defer GinkgoRecover()
			// Accept a single connection
			defer ln.Close()
			conn, err := ln.Accept()
			Expect(err).ToNot(HaveOccurred())
			defer conn.Close()
			if in != "" {
				buf := make([]byte, len(in))
				_, err := io.ReadFull(conn, buf)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(in))
			}
			if out != "" {
				_, err := io.WriteString(conn, out)
				Expect(err).ToNot(HaveOccurred())
			}
		}()
		Expect(err).ToNot(HaveOccurred())
		return ln.Addr().(*net.TCPAddr)
	}

	var t *Tunnel

	BeforeEach(func() {
		t = new(Tunnel)
		Expect(t).NotTo(BeNil())
	})

	It("should make a connection", func() {
		done := make(chan struct{})
		lAddr := listen(done, "", "")
		err := t.create(nil, nil, lAddr.Port)
		Expect(err).ToNot(HaveOccurred())

		conn, err := net.DialTCP("tcp", nil, lAddr)
		Expect(err).ToNot(HaveOccurred())

		defer conn.Close()
		Eventually(done).Should(BeClosed())
		Expect(t.Close()).To(Succeed())
	})

	It("should pass data through", func() {
		done := make(chan struct{})
		testIn := "input test string"
		testOut := "output test string"
		lAddr := listen(done, testIn, testOut)
		err := t.create(nil, nil, lAddr.Port)
		Expect(err).ToNot(HaveOccurred())
		conn, err := net.DialTCP("tcp", nil, t.Addr().(*net.TCPAddr))
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		_, err = io.WriteString(conn, testIn)
		Expect(err).ToNot(HaveOccurred())
		buf := make([]byte, len(testOut))
		_, err = io.ReadFull(conn, buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(buf)).To(Equal(testOut))

		Eventually(done).Should(BeClosed())
		Expect(t.Close()).To(Succeed())
	})

	It("should fail to listen on invalid address", func() {
		err := t.Create(invalidIP, nil)
		Expect(err).To(HaveOccurred())
	})

	It("should fail to connect to invalid address", func() {
		err := t.create(nil, nil, 0)
		Expect(err).ToNot(HaveOccurred())

		conn, err := net.DialTCP("tcp", nil, t.Addr().(*net.TCPAddr))
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		// There is no good way to check for closed connection
		// so we wait for some error to occur:
		// write tcp [::1]:36454->[::1]:42173: write: broken pipe
		// It usually takes 2 attempts.
		Eventually(func() error {
			_, err = io.WriteString(conn, "test")
			return err
		}).Should(HaveOccurred())
	})
})
