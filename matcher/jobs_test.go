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
	"crypto/rsa"
	"errors"
	"net"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/tunnels"
	"git.tizen.org/tools/boruta/workers"

	gomock "github.com/golang/mock/gomock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Jobs", func() {
	Describe("NewJobsManager", func() {
		It("should init all fields", func() {
			w := &MockWorkersManager{}

			jm := NewJobsManager(w)
			Expect(jm).NotTo(BeNil())
			Expect(jm.(*JobsManagerImpl).jobs).NotTo(BeNil())
			Expect(jm.(*JobsManagerImpl).workers).To(Equal(w))
			Expect(jm.(*JobsManagerImpl).mutex).NotTo(BeNil())
			Expect(jm.(*JobsManagerImpl).newTunnel).NotTo(BeNil())
			Expect(jm.(*JobsManagerImpl).newTunnel()).NotTo(BeNil())
		})
		It("should not use workers", func() {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()

			w := NewMockWorkersManager(ctrl)

			NewJobsManager(w)
		})
	})
	Describe("With prepared job data", func() {
		var (
			ctrl    *gomock.Controller
			w       *MockWorkersManager
			ttm     *MockTunneler
			jm      JobsManager
			ip      net.IP         = net.IPv4(5, 6, 7, 8)
			key     rsa.PrivateKey = rsa.PrivateKey{}
			addr    net.Addr       = &net.TCPAddr{IP: net.IPv4(10, 11, 12, 13), Port: 12345}
			req     ReqID          = ReqID(67)
			worker  WorkerUUID     = WorkerUUID("TestWorker")
			testerr error          = errors.New("TestError")
		)
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			w = NewMockWorkersManager(ctrl)
			ttm = NewMockTunneler(ctrl)
			jm = NewJobsManager(w)
			jm.(*JobsManagerImpl).newTunnel = func() tunnels.Tunneler { return ttm }
		})
		AfterEach(func() {
			ctrl.Finish()
		})
		assertJob := func(job *workers.Job) {
			Expect(job.Access.Addr).To(Equal(addr))
			Expect(job.Access.Key).To(Equal(key))
			Expect(job.Req).To(Equal(req))
		}
		Describe("Create", func() {
			It("should create a new, properly initialized job", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(key, nil),
					ttm.EXPECT().Create(nil, ip).Return(nil),
					ttm.EXPECT().Addr().Return(addr),
				)

				err := jm.Create(req, worker)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(jm.(*JobsManagerImpl).jobs)).To(Equal(1))
				job, ok := jm.(*JobsManagerImpl).jobs[worker]
				Expect(ok).To(BeTrue())
				assertJob(job)
			})
			It("should fail to create another job for same worker", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(key, nil),
					ttm.EXPECT().Create(nil, ip).Return(nil),
					ttm.EXPECT().Addr().Return(addr),
				)

				// Create first job.
				err := jm.Create(req, worker)
				Expect(err).NotTo(HaveOccurred())

				// Create another job for the same worker.
				err = jm.Create(req, worker)
				Expect(err).To(Equal(ErrJobAlreadyExists))
			})
			It("should fail when GetWorkerIP fails", func() {
				w.EXPECT().GetWorkerIP(worker).Return(nil, testerr)

				err := jm.Create(req, worker)
				Expect(err).To(Equal(testerr))
			})
			It("should fail and close tunnel when GetWorkerKey fails", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(rsa.PrivateKey{}, testerr),
				)

				err := jm.Create(req, worker)
				Expect(err).To(Equal(testerr))
			})
			It("should fail when tunnel creation fails", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(key, nil),
					ttm.EXPECT().Create(nil, ip).Return(testerr),
				)

				err := jm.Create(req, worker)
				Expect(err).To(Equal(testerr))
			})
		})
		Describe("Get", func() {
			It("should get existing job", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(key, nil),
					ttm.EXPECT().Create(nil, ip).Return(nil),
					ttm.EXPECT().Addr().Return(addr),
				)

				err := jm.Create(req, worker)
				Expect(err).NotTo(HaveOccurred())

				job, err := jm.Get(worker)
				Expect(err).NotTo(HaveOccurred())

				assertJob(job)
			})
			It("should fail getting nonexistent job", func() {
				job, err := jm.Get(worker)
				Expect(err).To(Equal(NotFoundError("Job")))
				Expect(job).To(BeNil())
			})
		})
		Describe("Finish", func() {
			It("should finish existing job", func() {
				gomock.InOrder(
					w.EXPECT().GetWorkerIP(worker).Return(ip, nil),
					w.EXPECT().GetWorkerKey(worker).Return(key, nil),
					ttm.EXPECT().Create(nil, ip).Return(nil),
					ttm.EXPECT().Addr().Return(addr),
					ttm.EXPECT().Close(),
					w.EXPECT().PrepareWorker(worker, true),
				)

				err := jm.Create(req, worker)
				Expect(err).NotTo(HaveOccurred())

				err = jm.Finish(worker)
				Expect(err).NotTo(HaveOccurred())

				Expect(jm.(*JobsManagerImpl).jobs).To(BeEmpty())
			})
			It("should fail to finish nonexistent job", func() {
				w.EXPECT().PrepareWorker(worker, true)
				err := jm.Finish(worker)
				Expect(err).To(Equal(NotFoundError("Job")))
			})
		})
	})
})
