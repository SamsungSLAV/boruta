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

package requests

//go:generate mockgen -package requests -destination=workersmanager_mock_test.go -write_package_comment=false git.tizen.org/tools/boruta/matcher WorkersManager
//go:generate mockgen -package requests -destination=requests/jobsmanager_mock_test.go -write_package_comment=false git.tizen.org/tools/boruta/matcher JobsManager

import (
	"errors"
	"time"

	"git.tizen.org/tools/boruta"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Requests as RequestsManager", func() {
	Describe("With RequestsManager created", func() {
		var ctrl *gomock.Controller
		var wm *MockWorkersManager
		var jm *MockJobsManager
		var R *ReqsCollection
		testErr := errors.New("Test Error")

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			wm = NewMockWorkersManager(ctrl)
			jm = NewMockJobsManager(ctrl)
			wm.EXPECT().SetChangeListener(gomock.Any())
			R = NewRequestQueue(wm, jm)
		})
		AfterEach(func() {
			R.Finish()
			ctrl.Finish()
		})

		Describe("Iterations", func() {
			var entered chan int
			testMutex := func() {
				R.mutex.Lock()
				defer R.mutex.Unlock()
				entered <- 1
			}
			BeforeEach(func() {
				entered = make(chan int)
			})
			Describe("InitIteration", func() {
				It("should init iterations and lock requests mutex", func() {
					err := R.InitIteration()
					Expect(err).NotTo(HaveOccurred())
					Expect(R.iterating).To(BeTrue())

					// Verify that mutex is locked.
					go testMutex()
					Consistently(entered).ShouldNot(Receive())

					// Release the mutex
					R.mutex.Unlock()
					Eventually(entered).Should(Receive())
				})
				It("should return error and remain mutex unlocked if iterations are already started", func() {
					R.mutex.Lock()
					R.iterating = true
					R.mutex.Unlock()

					err := R.InitIteration()
					Expect(err).To(Equal(boruta.ErrInternalLogicError))

					// Verify that mutex is not locked.
					go testMutex()
					Eventually(entered).Should(Receive())
				})
			})
			Describe("TerminateIteration", func() {
				It("should terminate iterations and unlock requests mutex", func() {
					err := R.InitIteration()
					Expect(err).NotTo(HaveOccurred())

					R.TerminateIteration()
					Expect(R.iterating).To(BeFalse())

					// Verify that mutex is not locked.
					go testMutex()
					Eventually(entered).Should(Receive())
				})
				It("should just release mutex if iterations are not started", func() {
					R.mutex.Lock()

					R.TerminateIteration()

					// Verify that mutex is not locked.
					go testMutex()
					Eventually(entered).Should(Receive())
				})
			})
		})
		Describe("Iterating over requests", func() {
			verify := []boruta.ReqID{3, 5, 1, 2, 7, 4, 6}
			BeforeEach(func() {
				now := time.Now()
				tomorrow := now.AddDate(0, 0, 1)
				wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), testErr).AnyTimes()
				insert := func(p boruta.Priority) {
					_, err := R.NewRequest(boruta.Capabilities{}, p, boruta.UserInfo{}, now, tomorrow)
					Expect(err).NotTo(HaveOccurred())
				}
				insert(3) //1
				insert(3) //2
				insert(1) //3
				insert(5) //4
				insert(1) //5
				insert(5) //6
				insert(3) //7
			})
			It("should properly iterate over requests", func() {
				reqs := make([]boruta.ReqID, 0)

				R.InitIteration()
				for r, ok := R.Next(); ok; r, ok = R.Next() {
					reqs = append(reqs, r)
				}
				R.TerminateIteration()

				Expect(reqs).To(Equal(verify))
			})
			It("should restart iterations in new critical section", func() {
				for times := 0; times < len(verify); times++ {
					reqs := make([]boruta.ReqID, 0)
					i := 0
					R.InitIteration()
					for r, ok := R.Next(); ok && i < times; r, ok = R.Next() {
						reqs = append(reqs, r)
						i++
					}
					R.TerminateIteration()
					Expect(reqs).To(Equal(verify[:times]))
				}
			})
			It("should panic if Next is called without InitIteration", func() {
				wrap := func() {
					R.mutex.Lock()
					defer R.mutex.Unlock()
					R.Next()
				}
				Expect(wrap).To(Panic())
			})
		})
		Describe("With request in the queue", func() {
			var now, tomorrow time.Time
			var req, noreq boruta.ReqID
			var rinfo *boruta.ReqInfo
			BeforeEach(func() {
				now = time.Now()
				tomorrow = now.AddDate(0, 0, 1)
				wm.EXPECT().TakeBestMatchingWorker(gomock.Any(), gomock.Any()).Return(boruta.WorkerUUID(""), testErr).AnyTimes()
				var err error
				req, err = R.NewRequest(boruta.Capabilities{}, 3, boruta.UserInfo{}, now, tomorrow)
				Expect(err).NotTo(HaveOccurred())
				var ok bool
				R.mutex.Lock()
				rinfo, ok = R.requests[req]
				R.mutex.Unlock()
				Expect(ok).To(BeTrue())
				noreq = req + 1
			})
			Describe("VerifyIfReady", func() {
				It("should fail if reqID is unknown", func() {
					Expect(R.VerifyIfReady(noreq, now)).To(BeFalse())
				})
				It("should fail if state is not WAIT", func() {
					states := []boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID,
						boruta.DONE, boruta.FAILED}
					for _, s := range states {
						R.mutex.Lock()
						rinfo.State = s
						R.mutex.Unlock()
						Expect(R.VerifyIfReady(req, now)).To(BeFalse(), "state = %v", s)
					}
				})
				It("should fail if Deadline is reached or passed", func() {
					Expect(R.VerifyIfReady(req, tomorrow.Add(-time.Hour))).To(BeTrue())
					Expect(R.VerifyIfReady(req, tomorrow)).To(BeFalse())
					Expect(R.VerifyIfReady(req, tomorrow.Add(time.Hour))).To(BeFalse())
				})
				It("should fail if ValidAfter is in future", func() {
					Expect(R.VerifyIfReady(req, now.Add(-time.Hour))).To(BeFalse())
					Expect(R.VerifyIfReady(req, now)).To(BeTrue())
					Expect(R.VerifyIfReady(req, now.Add(time.Hour))).To(BeTrue())
				})
				It("should succeed if request is known, in WAIT state and now is between ValidAfter and Deadline", func() {
					Expect(R.VerifyIfReady(req, now.Add(12*time.Hour))).To(BeTrue())
				})
			})
			Describe("Get", func() {
				It("should fail if reqID is unknown", func() {
					r, err := R.Get(noreq)
					Expect(err).To(Equal(boruta.NotFoundError("Request")))
					Expect(r).To(Equal(boruta.ReqInfo{}))
				})
				It("should succeed if reqID is valid", func() {
					r, err := R.Get(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(r).To(Equal(*rinfo))
				})
			})
			Describe("Timeout", func() {
				It("should fail if reqID is unknown", func() {
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Timeout(noreq)
					Expect(err).To(Equal(boruta.NotFoundError("Request")))
					Expect(R.queue.length).To(Equal(uint(1)))
				})
				It("should fail if request is not in WAIT state", func() {
					R.mutex.Lock()
					rinfo.Deadline = now.Add(-time.Hour)
					R.mutex.Unlock()
					Expect(R.queue.length).To(Equal(uint(1)))
					states := []boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID,
						boruta.DONE, boruta.FAILED}
					for _, s := range states {
						R.mutex.Lock()
						rinfo.State = s
						R.mutex.Unlock()
						err := R.Timeout(req)
						Expect(err).To(Equal(ErrModificationForbidden), "state = %v", s)
						Expect(R.queue.length).To(Equal(uint(1)), "state = %v", s)
					}
				})
				It("should fail if deadline is in the future", func() {
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Timeout(req)
					Expect(err).To(Equal(ErrModificationForbidden))
					Expect(R.queue.length).To(Equal(uint(1)))
				})
				It("should pass if deadline is past", func() {
					R.mutex.Lock()
					rinfo.Deadline = now.Add(-time.Hour)
					R.mutex.Unlock()
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Timeout(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(rinfo.State).To(Equal(boruta.TIMEOUT))
					Expect(R.queue.length).To(BeZero())
				})
			})
			Describe("Close", func() {
				It("should fail if reqID is unknown", func() {
					err := R.Close(noreq)
					Expect(err).To(Equal(boruta.NotFoundError("Request")))
				})
				It("should fail if request is not in INPROGRESS state", func() {
					states := []boruta.ReqState{boruta.WAIT, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID,
						boruta.DONE, boruta.FAILED}
					for _, state := range states {
						R.mutex.Lock()
						rinfo.State = state
						R.mutex.Unlock()

						err := R.Close(req)
						Expect(err).To(Equal(ErrModificationForbidden), "state = %s", state)
					}
				})
				It("should fail if request has no job assigned", func() {
					R.mutex.Lock()
					rinfo.State = boruta.INPROGRESS
					Expect(rinfo.Job).To(BeNil())
					R.mutex.Unlock()

					err := R.Close(req)
					Expect(err).To(Equal(boruta.ErrInternalLogicError))
				})
				It("should fail if job's is not yet timed out", func() {
					R.mutex.Lock()
					rinfo.State = boruta.INPROGRESS
					rinfo.Job = &boruta.JobInfo{
						Timeout: time.Now().AddDate(0, 0, 1),
					}
					R.mutex.Unlock()

					err := R.Close(req)
					Expect(err).To(Equal(ErrModificationForbidden))
				})
				It("should close request and release worker", func() {
					testWorker := boruta.WorkerUUID("TestWorker")
					R.mutex.Lock()
					rinfo.State = boruta.INPROGRESS
					rinfo.Job = &boruta.JobInfo{
						Timeout:    time.Now().AddDate(0, 0, -1),
						WorkerUUID: testWorker,
					}
					R.mutex.Unlock()
					gomock.InOrder(
						jm.EXPECT().Finish(testWorker, true),
					)
					err := R.Close(req)
					Expect(err).NotTo(HaveOccurred())
					Expect(rinfo.State).To(Equal(boruta.DONE))
				})
			})
			Describe("Run", func() {
				testWorker := boruta.WorkerUUID("TestWorker")

				It("should fail if reqID is unknown", func() {
					R.mutex.Lock()
					defer R.mutex.Unlock()
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Run(noreq, testWorker)
					Expect(err).To(Equal(boruta.NotFoundError("Request")))
					Expect(R.queue.length).To(Equal(uint(1)))
				})
				It("should fail if reqID is unknown during iteration", func() {
					R.InitIteration()
					defer R.TerminateIteration()
					Expect(R.iterating).To(BeTrue())
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Run(noreq, testWorker)
					Expect(err).To(Equal(boruta.NotFoundError("Request")))
					Expect(R.iterating).To(BeTrue())
					Expect(R.queue.length).To(Equal(uint(1)))
				})
				It("should fail if request is not in WAIT state", func() {
					states := []boruta.ReqState{boruta.INPROGRESS, boruta.CANCEL, boruta.TIMEOUT, boruta.INVALID,
						boruta.DONE, boruta.FAILED}
					for _, state := range states {
						R.InitIteration()
						Expect(R.queue.length).To(Equal(uint(1)), "state = %s", state)
						rinfo.State = state
						err := R.Run(req, boruta.WorkerUUID("TestWorker"))
						Expect(err).To(Equal(ErrModificationForbidden), "state = %s", state)
						Expect(R.queue.length).To(Equal(uint(1)), "state = %s", state)
						R.TerminateIteration()
					}
				})
				It("should start progress for valid reqID", func() {
					R.InitIteration()
					defer R.TerminateIteration()
					Expect(R.queue.length).To(Equal(uint(1)))
					err := R.Run(req, testWorker)
					Expect(err).NotTo(HaveOccurred())
					Expect(rinfo.State).To(Equal(boruta.INPROGRESS))
					Expect(rinfo.Job.Timeout).To(BeTemporally(">", time.Now()))
					Expect(R.queue.length).To(BeZero())
				})
				It("should start progress and break iterations when iterating", func() {
					R.InitIteration()
					defer R.TerminateIteration()
					Expect(R.queue.length).To(Equal(uint(1)))
					Expect(R.iterating).To(BeTrue())
					err := R.Run(req, testWorker)
					Expect(err).NotTo(HaveOccurred())
					Expect(rinfo.State).To(Equal(boruta.INPROGRESS))
					Expect(rinfo.Job.Timeout).To(BeTemporally(">", time.Now()))
					Expect(R.iterating).To(BeFalse())
					Expect(R.queue.length).To(BeZero())
				})
			})
		})
	})
})
