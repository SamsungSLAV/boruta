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

package dryad

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net"
	"net/rpc"

	"github.com/SamsungSLAV/boruta/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
)

var _ = Describe("Dryad", func() {
	var ctrl *gomock.Controller
	var md *mocks.MockDryad

	const badAddr = "1500.100.900.300:400"
	const testMessage = "This is a test message. Do not panic!"
	var testError = errors.New("test error")

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		md = mocks.NewMockDryad(ctrl)
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("DryadService creation", func() {
		It("NewDryadService should create new service", func() {
			service := NewDryadService(md)
			Expect(service).NotTo(BeNil())
			Expect(service.impl).To(Equal(md))
		})
		It("RegisterDryadService should create and register new service", func() {
			rpcServer := rpc.NewServer()

			err := RegisterDryadService(rpcServer, md)
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("with DryadService listening", func() {
		var listener net.Listener

		BeforeEach(func() {
			var err error
			listener, err = net.Listen("tcp", "")
			Expect(err).NotTo(HaveOccurred())
			rpcServer := rpc.NewServer()
			err = RegisterDryadService(rpcServer, md)
			Expect(err).NotTo(HaveOccurred())
			go rpcServer.Accept(listener)
		})
		AfterEach(func() {
			err := listener.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Client connection", func() {
			It("DialDryadClient should connect to the service", func() {
				client, err := DialDryadClient(listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).NotTo(BeNil())
			})
			It("DialDryadClient should not connect to not existing address", func() {
				client, err := DialDryadClient(badAddr)
				Expect(err).To(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).To(BeNil())
			})
			It("NewDryadClient should create new client", func() {
				rpcClient, err := rpc.Dial("tcp", listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())

				client := NewDryadClient(rpcClient)
				Expect(client).NotTo(BeNil())
				Expect(client.client).To(Equal(rpcClient))
			})
			It("Close should close connection created with DialDryadClient", func() {
				client, err := DialDryadClient(listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).NotTo(BeNil())

				err = client.Close()
				Expect(err).NotTo(HaveOccurred())
			})
			It("Create should connect to the service", func() {
				client := new(DryadClient)
				addr, err := net.ResolveTCPAddr("tcp", listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())

				err = client.Create(addr)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).NotTo(BeNil())
			})
			It("Create should fail to connect to not existing address", func() {
				client := new(DryadClient)
				badTCPAddr := &net.TCPAddr{
					IP:   net.IPv6unspecified,
					Port: -1000,
				}

				err := client.Create(badTCPAddr)
				Expect(err).To(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).To(BeNil())
			})
			It("Close should close connection created with DialDryadClient", func() {
				client := new(DryadClient)
				addr, err := net.ResolveTCPAddr("tcp", listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())
				err = client.Create(addr)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).NotTo(BeNil())

				err = client.Close()
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Describe("with DryadClient connected", func() {
			var client *DryadClient

			BeforeEach(func() {
				client = new(DryadClient)
				addr, err := net.ResolveTCPAddr("tcp", listener.Addr().String())
				Expect(err).NotTo(HaveOccurred())
				err = client.Create(addr)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
				Expect(client.client).NotTo(BeNil())
			})
			AfterEach(func() {
				err := client.Close()
				Expect(err).NotTo(HaveOccurred())
			})

			Describe("Healthcheck", func() {
				It("should run Healthcheck on server", func() {
					md.EXPECT().Healthcheck()

					err := client.Healthcheck()
					Expect(err).NotTo(HaveOccurred())
				})
				It("should run Healthcheck on server and pass an error", func() {
					md.EXPECT().Healthcheck().Return(testError)

					err := client.Healthcheck()
					Expect(err).To(Equal(rpc.ServerError(testError.Error())))
				})
			})
			Describe("PutInMaintenance", func() {
				It("should run PutInMaintenance on server", func() {
					md.EXPECT().PutInMaintenance(testMessage)

					err := client.PutInMaintenance(testMessage)
					Expect(err).NotTo(HaveOccurred())
				})
				It("should run PutInMaintenance on server and pass an error", func() {
					md.EXPECT().PutInMaintenance(testMessage).Return(testError)

					err := client.PutInMaintenance(testMessage)
					Expect(err).To(Equal(rpc.ServerError(testError.Error())))
				})
			})
			Describe("Prepare", func() {
				const bits = 32

				It("should run Prepare on server", func() {
					privateKey, err := rsa.GenerateKey(rand.Reader, bits)
					Expect(err).NotTo(HaveOccurred())
					publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
					Expect(err).NotTo(HaveOccurred())
					md.EXPECT().Prepare(gomock.Any()).DoAndReturn(func(key *ssh.PublicKey) error {
						received := (*key).Marshal()
						expected := publicKey.Marshal()
						Expect(received).To(Equal(expected))
						return nil
					})

					err = client.Prepare(&publicKey)
					Expect(err).NotTo(HaveOccurred())
				})
				It("should run Prepare on server and pass an error", func() {
					privateKey, err := rsa.GenerateKey(rand.Reader, bits)
					Expect(err).NotTo(HaveOccurred())
					publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
					Expect(err).NotTo(HaveOccurred())
					md.EXPECT().Prepare(gomock.Any()).DoAndReturn(func(key *ssh.PublicKey) error {
						received := (*key).Marshal()
						expected := publicKey.Marshal()
						Expect(received).To(Equal(expected))
						return testError
					})

					err = client.Prepare(&publicKey)
					Expect(err).To(Equal(rpc.ServerError(testError.Error())))
				})
			})
		})
	})
})
