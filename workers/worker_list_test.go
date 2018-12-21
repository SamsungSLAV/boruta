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

package workers

//go:generate mockgen -package workers -destination=dryadclientmanager_mock_test.go -write_package_comment=false -mock_names ClientManager=MockDryadClientManager github.com/SamsungSLAV/boruta/rpc/dryad ClientManager

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	"github.com/SamsungSLAV/boruta/rpc/dryad"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/satori/go.uuid"
)

var _ = Describe("WorkerList", func() {
	var wl *WorkerList
	dryadAddr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 7175,
	}
	sshdAddr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 22,
	}
	missingPort := &net.TCPAddr{
		IP: dryadAddr.IP,
	}
	BeforeEach(func() {
		sizeRSA = 256
		wl = NewWorkerList()
	})

	It("should return non-nil new DryadClient every time called", func() {
		for i := 0; i < 3; i++ {
			Expect(wl.newDryadClient()).NotTo(BeNil(), "i = %d", i)
		}
	})

	getUUID := func() string {
		u, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return u.String()
	}

	Describe("Register", func() {
		var registeredWorkers []string
		invalidAddr := "addr.invalid"

		BeforeEach(func() {
			registeredWorkers = make([]string, 0)
		})

		compareLists := func() {
			wl.mutex.RLock()
			defer wl.mutex.RUnlock()
			// Check if all registeredWorkers are present
			for _, uuid := range registeredWorkers {
				_, ok := wl.workers[boruta.WorkerUUID(uuid)]
				Expect(ok).To(BeTrue())
			}
			// Check if all workers from the wl.workers are present
			for _, workerInfo := range wl.workers {
				ok := false
				for _, uuid := range registeredWorkers {
					if workerInfo.WorkerUUID == boruta.WorkerUUID(uuid) {
						ok = true
						break
					}
				}
				Expect(ok).To(BeTrue())
			}
		}

		It("should fail if UUID is not present", func() {
			err := wl.Register(nil, "", "")
			Expect(err).To(Equal(ErrMissingUUID))
		})

		getRandomCaps := func() boruta.Capabilities {
			return boruta.Capabilities{
				UUID: getUUID(),
			}
		}

		DescribeTable("dryad and sshd addresses",
			func(dryadAddress, sshAddress string, errMatcher types.GomegaMatcher) {
				caps := getRandomCaps()
				err := wl.Register(caps, dryadAddress, sshAddress)
				Expect(err).To(errMatcher)
			},
			Entry("both addresses missing", "", "", Equal(ErrMissingIP)),
			Entry("sshd address missing", dryadAddr.String(), "", Equal(ErrMissingIP)),
			Entry("dryad address missing", "", sshdAddr.String(), Equal(ErrMissingIP)),
			Entry("dryad port missing", missingPort.String(), sshdAddr.String(), Equal(ErrMissingPort)),
			Entry("sshd port missing", dryadAddr.String(), missingPort.String(), Equal(ErrMissingPort)),
			Entry("both ports missing", missingPort.String(), missingPort.String(), Equal(ErrMissingPort)),
			Entry("both invalid", invalidAddr, invalidAddr, HaveOccurred()),
			Entry("dryad invalid", invalidAddr, sshdAddr.String(), HaveOccurred()),
			Entry("sshd invalid", dryadAddr.String(), invalidAddr, HaveOccurred()),
		)

		It("should add Worker in MAINTENANCE state", func() {
			caps := getRandomCaps()
			err := wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			uuid := boruta.WorkerUUID(caps[UUID])
			wl.mutex.RLock()
			defer wl.mutex.RUnlock()
			Expect(wl.workers).To(HaveKey(uuid))
			Expect(wl.workers[uuid].State).To(Equal(boruta.MAINTENANCE))
			Expect(wl.workers[uuid].backgroundOperation).NotTo(BeNil())
		})

		It("should update the caps when called twice for the same worker", func() {
			var err error
			wl.mutex.RLock()
			Expect(wl.workers).To(BeEmpty())
			wl.mutex.RUnlock()
			caps := getRandomCaps()

			By("registering worker")
			err = wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps[UUID])
			compareLists()

			By("updating the caps")
			caps["test-key"] = "test-value"
			err = wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			wl.mutex.RLock()
			Expect(wl.workers[boruta.WorkerUUID(caps[UUID])].Caps).To(Equal(caps))
			wl.mutex.RUnlock()
			compareLists()
		})

		It("should work when called once", func() {
			var err error
			wl.mutex.RLock()
			Expect(wl.workers).To(BeEmpty())
			wl.mutex.RUnlock()
			caps := getRandomCaps()

			By("registering first worker")
			err = wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps[UUID])
			compareLists()
		})

		It("should work when called twice with different caps", func() {
			var err error
			wl.mutex.RLock()
			Expect(wl.workers).To(BeEmpty())
			wl.mutex.RUnlock()
			caps1 := getRandomCaps()
			caps2 := getRandomCaps()

			By("registering first worker")
			err = wl.Register(caps1, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps1[UUID])
			compareLists()

			By("registering second worker")
			err = wl.Register(caps2, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps2[UUID])
			compareLists()
		})
	})

	Context("with worker registered", func() {
		var worker boruta.WorkerUUID

		randomUUID := func() boruta.WorkerUUID {
			newUUID := worker
			for newUUID == worker {
				newUUID = boruta.WorkerUUID(getUUID())
			}
			return newUUID
		}
		registerWorker := func() boruta.WorkerUUID {
			capsUUID := randomUUID()
			err := wl.Register(boruta.Capabilities{UUID: string(capsUUID)}, dryadAddr.String(), sshdAddr.String())
			Expect(err).ToNot(HaveOccurred())
			wl.mutex.RLock()
			Expect(wl.workers).ToNot(BeEmpty())
			wl.mutex.RUnlock()
			return capsUUID
		}

		BeforeEach(func() {
			wl.mutex.RLock()
			Expect(wl.workers).To(BeEmpty())
			wl.mutex.RUnlock()
			worker = registerWorker()
		})

		AfterEach(func() {
			wl.Deregister(worker)
		})

		Describe("SetFail", func() {
			It("should fail to SetFail of nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.SetFail(uuid, "")
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to SetFail", func() {
				for _, state := range []boruta.WorkerState{boruta.IDLE, boruta.RUN, boruta.PREPARE} {
					wl.mutex.Lock()
					wl.workers[worker].State = state
					wl.mutex.Unlock()
					err := wl.SetFail(worker, "")
					Expect(err).ToNot(HaveOccurred())
					wl.mutex.RLock()
					Expect(wl.workers[worker].State).To(Equal(boruta.FAIL))
					wl.mutex.RUnlock()
				}
			})

			It("Should fail to SetFail in MAINTENANCE or BUSY state", func() {
				for _, state := range []boruta.WorkerState{boruta.MAINTENANCE, boruta.BUSY} {
					wl.mutex.Lock()
					wl.workers[worker].State = state
					wl.mutex.Unlock()
					err := wl.SetFail(worker, "")
					Expect(err).To(Equal(ErrInMaintenance))
					wl.mutex.RLock()
					Expect(wl.workers[worker].State).To(Equal(state))
					wl.mutex.RUnlock()
				}
			})
		})

		Describe("Deregister", func() {
			It("should fail to deregister nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.Deregister(uuid)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to deregister worker in MAINTENANCE state", func() {
				err := wl.Deregister(worker)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers).ToNot(HaveKey(worker))
				wl.mutex.RUnlock()
			})

			It("should work to deregister worker in FAIL state", func() {
				wl.mutex.Lock()
				wl.workers[worker].State = boruta.FAIL
				wl.mutex.Unlock()
				err := wl.Deregister(worker)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers).ToNot(HaveKey(worker))
				wl.mutex.RUnlock()
			})

			It("should fail to deregister same worker twice", func() {
				err := wl.Deregister(worker)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers).ToNot(HaveKey(worker))
				wl.mutex.RUnlock()

				err = wl.Deregister(worker)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should fail to deregister worker not in FAIL or MAINTENANCE state", func() {
				for _, state := range []boruta.WorkerState{boruta.IDLE, boruta.RUN, boruta.PREPARE, boruta.BUSY} {
					wl.mutex.Lock()
					wl.workers[worker].State = state
					wl.mutex.Unlock()
					err := wl.Deregister(worker)
					Expect(err).To(Equal(ErrNotInFailOrMaintenance))
					wl.mutex.RLock()
					Expect(wl.workers).To(HaveKey(worker))
					wl.mutex.RUnlock()
				}
			})
		})

		Describe("with dryadClientManager mockup", func() {
			var ctrl *gomock.Controller
			var dcm *MockDryadClientManager
			ip := net.IPv4(2, 4, 6, 8)
			testerr := errors.New("Test Error")
			var info *mapWorker
			noWorker := boruta.WorkerUUID("There's no such worker")
			putStr := "maintenance"

			eventuallyState := func(info *mapWorker, state boruta.WorkerState) {
				EventuallyWithOffset(1, func() boruta.WorkerState {
					wl.mutex.RLock()
					defer wl.mutex.RUnlock()
					return info.State
				}).Should(Equal(state))
			}
			eventuallyKey := func(info *mapWorker, match types.GomegaMatcher) {
				EventuallyWithOffset(1, func() *rsa.PrivateKey {
					wl.mutex.RLock()
					defer wl.mutex.RUnlock()
					return info.key
				}).Should(match)
			}

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				dcm = NewMockDryadClientManager(ctrl)
				wl.newDryadClient = func() dryad.ClientManager {
					return dcm
				}

				var ok bool
				wl.mutex.Lock()
				info, ok = wl.workers[worker]
				Expect(ok).To(BeTrue())
				Expect(info.key).To(BeNil())
				info.dryad = new(net.TCPAddr)
				info.dryad.IP = ip
				wl.mutex.Unlock()
			})

			AfterEach(func() {
				ctrl.Finish()
			})

			Describe("SetState", func() {
				It("should fail to SetState of nonexistent worker", func() {
					uuid := randomUUID()
					err := wl.SetState(uuid, boruta.MAINTENANCE)
					Expect(err).To(Equal(ErrWorkerNotFound))
				})

				It("should fail to SetState for invalid transitions", func() {
					invalidTransitions := [][]boruta.WorkerState{
						{boruta.RUN, boruta.IDLE},
						{boruta.FAIL, boruta.IDLE},
						{boruta.BUSY, boruta.IDLE},
					}
					for _, transition := range invalidTransitions {
						fromState, toState := transition[0], transition[1]
						wl.mutex.Lock()
						wl.workers[worker].State = fromState
						wl.mutex.Unlock()
						err := wl.SetState(worker, toState)
						Expect(err).To(Equal(ErrForbiddenStateChange))
						wl.mutex.RLock()
						Expect(wl.workers[worker].State).To(Equal(fromState))
						wl.mutex.RUnlock()
					}
				})

				It("should ignore SetState if transition is already ongoing", func() {
					invalidTransitions := [][]boruta.WorkerState{
						{boruta.PREPARE, boruta.IDLE},
						{boruta.BUSY, boruta.MAINTENANCE},
					}
					for _, transition := range invalidTransitions {
						fromState, toState := transition[0], transition[1]
						wl.mutex.Lock()
						wl.workers[worker].State = fromState
						wl.mutex.Unlock()
						err := wl.SetState(worker, toState)
						Expect(err).NotTo(HaveOccurred())
						wl.mutex.RLock()
						Expect(wl.workers[worker].State).To(Equal(fromState))
						wl.mutex.RUnlock()
					}
				})

				It("should fail to SetState for incorrect state argument", func() {
					invalidArgument := [][]boruta.WorkerState{
						{boruta.MAINTENANCE, boruta.RUN},
						{boruta.MAINTENANCE, boruta.FAIL},
						{boruta.MAINTENANCE, boruta.PREPARE},
						{boruta.MAINTENANCE, boruta.BUSY},
						{boruta.IDLE, boruta.FAIL},
						{boruta.IDLE, boruta.RUN},
						{boruta.IDLE, boruta.PREPARE},
						{boruta.IDLE, boruta.BUSY},
						{boruta.RUN, boruta.FAIL},
						{boruta.RUN, boruta.PREPARE},
						{boruta.RUN, boruta.BUSY},
						{boruta.FAIL, boruta.RUN},
						{boruta.FAIL, boruta.PREPARE},
						{boruta.FAIL, boruta.BUSY},
						{boruta.PREPARE, boruta.FAIL},
						{boruta.PREPARE, boruta.RUN},
						{boruta.PREPARE, boruta.BUSY},
						{boruta.BUSY, boruta.FAIL},
						{boruta.BUSY, boruta.RUN},
						{boruta.BUSY, boruta.PREPARE},
					}
					for _, transition := range invalidArgument {
						fromState, toState := transition[0], transition[1]
						wl.mutex.Lock()
						wl.workers[worker].State = fromState
						wl.mutex.Unlock()
						err := wl.SetState(worker, toState)
						Expect(err).To(Equal(ErrWrongStateArgument))
						wl.mutex.RLock()
						Expect(wl.workers[worker].State).To(Equal(fromState))
						wl.mutex.RUnlock()
					}
				})

				Describe("from MAINTENANCE to IDLE", func() {
					BeforeEach(func() {
						wl.mutex.Lock()
						info.State = boruta.MAINTENANCE
						wl.mutex.Unlock()
					})

					It("should work to SetState", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.Any()).Return(nil),
							dcm.EXPECT().Close().Do(func() {
								wl.mutex.Lock()
								Expect(info.State).To(Equal(boruta.PREPARE))
								wl.mutex.Unlock()
							}),
						)

						err := wl.SetState(worker, boruta.IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, boruta.IDLE)
						eventuallyKey(info, Not(Equal(&rsa.PrivateKey{})))
					})

					It("should fail to SetState if dryadClientManager fails to prepare client", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.Any()).Return(testerr),
							dcm.EXPECT().Close().Do(func() {
								wl.mutex.Lock()
								Expect(info.State).To(Equal(boruta.PREPARE))
								wl.mutex.Unlock()
							}),
						)

						err := wl.SetState(worker, boruta.IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, boruta.FAIL)
						Expect(info.key).To(BeNil())
					})

					It("should fail to SetState if dryadClientManager fails to create client", func() {
						dcm.EXPECT().Create(info.dryad).DoAndReturn(func(*net.TCPAddr) error {
							wl.mutex.Lock()
							Expect(info.State).To(Equal(boruta.PREPARE))
							wl.mutex.Unlock()
							return testerr
						})

						err := wl.SetState(worker, boruta.IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, boruta.FAIL)
						Expect(info.key).To(BeNil())
					})
				})

				trigger := make(chan int, 1)

				setTrigger := func(val int) {
					trigger <- val
				}
				eventuallyTrigger := func(val int) {
					EventuallyWithOffset(1, trigger).Should(Receive(Equal(val)))
				}

				fromStates := []boruta.WorkerState{boruta.IDLE, boruta.RUN, boruta.FAIL, boruta.PREPARE}
				for _, from := range fromStates {
					Describe("from "+string(from)+" to MAINTENANCE", func() {
						BeforeEach(func() {
							wl.mutex.Lock()
							info.State = from
							wl.mutex.Unlock()
						})

						It("should work to SetState", func() {
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad),
								dcm.EXPECT().PutInMaintenance(putStr),
								dcm.EXPECT().Close().Do(func() {
									wl.mutex.Lock()
									Expect(info.State).To(Equal(boruta.BUSY))
									wl.mutex.Unlock()
								}),
							)

							err := wl.SetState(worker, boruta.MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyState(info, boruta.MAINTENANCE)
						})

						It("should fail to SetState if dryadClientManager fails to put dryad in maintenance state", func() {
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad),
								dcm.EXPECT().PutInMaintenance(putStr).Return(testerr),
								dcm.EXPECT().Close().Do(func() {
									wl.mutex.Lock()
									Expect(info.State).To(Equal(boruta.BUSY))
									wl.mutex.Unlock()
									setTrigger(1)
								}),
							)

							err := wl.SetState(worker, boruta.MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyTrigger(1)
							eventuallyState(info, boruta.FAIL)
						})

						It("should fail to SetState if dryadClientManager fails to create client", func() {
							dcm.EXPECT().Create(info.dryad).Return(testerr).Do(func(*net.TCPAddr) {
								wl.mutex.Lock()
								Expect(info.State).To(Equal(boruta.BUSY))
								wl.mutex.Unlock()
								setTrigger(2)
							})

							err := wl.SetState(worker, boruta.MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyTrigger(2)
							eventuallyState(info, boruta.FAIL)
						})
					})
				}
			})
			Describe("withBackgroundContext", func() {
				var bc, noWorkerBc backgroundContext
				var c chan boruta.WorkerState
				var testState boruta.WorkerState = boruta.RUN
				var opEmpty = pendingOperation{}
				var opClosed = pendingOperation{got: true}
				var opState = pendingOperation{got: true, open: true, state: testState}

				BeforeEach(func() {
					c = make(chan boruta.WorkerState, backgroundOperationsBufferSize)
					bc = backgroundContext{
						c:    c,
						uuid: worker,
					}
					noWorkerBc = backgroundContext{
						c:    c,
						uuid: noWorker,
					}
				})

				AfterEach(func() {
					if c != nil {
						close(c)
					}
				})

				Describe("putInMaintenance", func() {
					It("should work", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr),
							dcm.EXPECT().Close(),
						)

						op, err := wl.putInMaintenance(bc)
						Expect(err).ToNot(HaveOccurred())
						Expect(op).To(Equal(opEmpty))
					})

					It("should fail if dryadClientManager fails to put dryad in maintenance state", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr).Return(testerr),
							dcm.EXPECT().Close(),
						)

						op, err := wl.putInMaintenance(bc)
						Expect(err).To(Equal(testerr))
						Expect(op).To(Equal(opEmpty))
					})

					It("should fail if dryadClientManager fails to create client", func() {
						dcm.EXPECT().Create(info.dryad).Return(testerr)

						op, err := wl.putInMaintenance(bc)
						Expect(err).To(Equal(testerr))
						Expect(op).To(Equal(opEmpty))
					})

					It("should fail if worker is not registered", func() {
						op, err := wl.putInMaintenance(noWorkerBc)
						Expect(err).To(Equal(ErrWorkerNotFound))
						Expect(op).To(Equal(opEmpty))
					})

					It("should break execution when channel is used before execution", func() {
						c <- testState

						op, err := wl.putInMaintenance(bc)
						Expect(err).ToNot(HaveOccurred())
						Expect(op).To(Equal(opState))
					})

					It("should break execution when channel is used during execution", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad).Do(func(*net.TCPAddr) {
								c <- testState
							}),
							dcm.EXPECT().Close(),
						)

						op, err := wl.putInMaintenance(bc)
						Expect(err).ToNot(HaveOccurred())
						Expect(op).To(Equal(opState))
					})

					It("should break execution when channel is closed before execution", func() {
						close(c)
						c = nil

						op, err := wl.putInMaintenance(bc)
						Expect(err).ToNot(HaveOccurred())
						Expect(op).To(Equal(opClosed))
					})

					It("should break execution when channel is closed during execution", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad).Do(func(*net.TCPAddr) {
								close(c)
								c = nil
							}),
							dcm.EXPECT().Close(),
						)

						op, err := wl.putInMaintenance(bc)
						Expect(err).ToNot(HaveOccurred())
						Expect(op).To(Equal(opClosed))
					})
				})

				Describe("prepareKeyAndSetState", func() {
					It("should ignore if worker is not registered", func() {
						op := wl.prepareKeyAndSetState(noWorkerBc)
						Expect(op).To(Equal(opEmpty))
					})

					It("should return immediately if new operation is pending on channel", func() {
						c <- testState
						op := wl.prepareKeyAndSetState(bc)
						Expect(op).To(Equal(opState))
					})

					It("should return immediately if channel is closed", func() {
						close(c)
						c = nil
						op := wl.prepareKeyAndSetState(bc)
						Expect(op).To(Equal(opClosed))
					})

					It("should return if new operation appeared on channel after prepareKey is completed", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.Any()),
							dcm.EXPECT().Close().Do(func() {
								c <- testState
							}),
						)

						op := wl.prepareKeyAndSetState(bc)
						Expect(op).To(Equal(opState))
					})

					It("should return if channel is closed after prepareKey is completed", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.Any()),
							dcm.EXPECT().Close().Do(func() {
								close(c)
								c = nil
							}),
						)

						op := wl.prepareKeyAndSetState(bc)
						Expect(op).To(Equal(opClosed))
					})
				})

				Describe("putInMaintenanceWorker", func() {
					It("should ignore if worker is not registered", func() {
						op := wl.putInMaintenanceWorker(noWorkerBc)
						Expect(op).To(Equal(opEmpty))
					})

					It("should return immediately if new operation is pending on channel", func() {
						c <- testState
						op := wl.putInMaintenanceWorker(bc)
						Expect(op).To(Equal(opState))
					})

					It("should return immediately if channel is closed", func() {
						close(c)
						c = nil
						op := wl.putInMaintenanceWorker(bc)
						Expect(op).To(Equal(opClosed))
					})

					It("should return if new operation appeared on channel after putInMaintenance is completed", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr),
							dcm.EXPECT().Close().Do(func() {
								c <- testState
							}),
						)

						op := wl.putInMaintenanceWorker(bc)
						Expect(op).To(Equal(opState))
					})

					It("should return if channel is closed after putInMaintenance is completed", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr),
							dcm.EXPECT().Close().Do(func() {
								close(c)
								c = nil
							}),
						)

						op := wl.putInMaintenanceWorker(bc)
						Expect(op).To(Equal(opClosed))
					})
				})

				Describe("checkPendingOperation", func() {
					It("should return got=false when nothing happens on channel", func() {
						op := checkPendingOperation(c)
						Expect(op).To(Equal(opEmpty))
					})

					It("should return state when new message appeared on channel", func() {
						c <- testState
						op := checkPendingOperation(c)
						Expect(op).To(Equal(opState))
					})

					It("should return last state when new message appeared on channel", func() {
						c <- boruta.FAIL
						c <- boruta.BUSY
						c <- boruta.FAIL
						c <- testState
						op := checkPendingOperation(c)
						Expect(op).To(Equal(opState))
					})

					It("should return open=false when channel is closed", func() {
						close(c)
						defer func() { c = nil }()
						op := checkPendingOperation(c)
						Expect(op).To(Equal(opClosed))
					})
				})

				Describe("backgroundLoop", func() {
					var done bool
					var m sync.Locker
					isDone := func() bool {
						m.Lock()
						defer m.Unlock()
						return done
					}
					setDone := func(val bool) {
						m.Lock()
						defer m.Unlock()
						done = val
					}
					run := func() {
						defer GinkgoRecover()
						wl.backgroundLoop(bc)
						setDone(true)
					}
					ignoredStates := []boruta.WorkerState{boruta.MAINTENANCE, boruta.IDLE, boruta.RUN, boruta.FAIL}

					BeforeEach(func() {
						m = new(sync.Mutex)
						setDone(false)
					})

					It("should run infinitely, but stop if channel is closed", func() {
						go run()
						Consistently(isDone).Should(BeFalse())
						close(c)
						c = nil
						Eventually(isDone).Should(BeTrue())
					})

					It("should ignore states not requiring any action and return after channel is closed", func() {
						go run()
						for _, state := range ignoredStates {
							By(string(state))
							c <- state
						}
						// Check that loop is still running.
						Consistently(isDone).Should(BeFalse())

						// Check loop returns after channel is closed.
						close(c)
						c = nil
						Eventually(isDone).Should(BeTrue())
					})

					It("should run prepareKey if PREPARE state is sent over channel", func() {
						go run()
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.Any()),
							dcm.EXPECT().Close().Do(func() {
								close(c)
								c = nil
							}),
						)
						c <- boruta.PREPARE
						Eventually(isDone).Should(BeTrue())
					})

					It("should run putInMaintenance if BUSY state is sent over channel", func() {
						go run()
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr),
							dcm.EXPECT().Close().Do(func() {
								close(c)
								c = nil
							}),
						)
						c <- boruta.BUSY
						Eventually(isDone).Should(BeTrue())
					})

					DescribeTable("should break running current background task if other state is sent over channel",
						func(current, another boruta.WorkerState) {
							go run()
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad).DoAndReturn(func(*net.TCPAddr) error {
									c <- another
									return nil
								}),
								dcm.EXPECT().Close().Do(func() {
									close(c)
									c = nil
								}),
							)
							c <- current
							Eventually(isDone).Should(BeTrue())
						},
						Entry("PREPARE->MAINTENANCE", boruta.PREPARE, boruta.MAINTENANCE),
						Entry("PREPARE->IDLE", boruta.PREPARE, boruta.IDLE),
						Entry("PREPARE->RUN", boruta.PREPARE, boruta.RUN),
						Entry("PREPARE->FAIL", boruta.PREPARE, boruta.FAIL),
						Entry("BUSY->MAINTENANCE", boruta.BUSY, boruta.MAINTENANCE),
						Entry("BUSY->IDLE", boruta.BUSY, boruta.IDLE),
						Entry("BUSY->RUN", boruta.BUSY, boruta.RUN),
						Entry("BUSY->FAIL", boruta.BUSY, boruta.FAIL),
					)

					DescribeTable("should break running current background task and start prepareKey if PREPARE is sent over channel",
						func(current boruta.WorkerState) {
							go run()
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad).DoAndReturn(func(*net.TCPAddr) error {
									c <- boruta.PREPARE
									return nil
								}),
								dcm.EXPECT().Close(),
								dcm.EXPECT().Create(info.dryad),
								dcm.EXPECT().Prepare(gomock.Any()),
								dcm.EXPECT().Close().Do(func() {
									close(c)
									c = nil
								}),
							)
							c <- current
							Eventually(isDone).Should(BeTrue())
						},
						Entry("PREPARE", boruta.PREPARE),
						Entry("BUSY", boruta.BUSY),
					)

					DescribeTable("should break running current background task and start putInMaintenance if BUSY is sent over channel",
						func(current boruta.WorkerState) {
							go run()
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad).DoAndReturn(func(*net.TCPAddr) error {
									c <- boruta.BUSY
									return nil
								}),
								dcm.EXPECT().Close(),
								dcm.EXPECT().Create(info.dryad),
								dcm.EXPECT().PutInMaintenance(putStr),
								dcm.EXPECT().Close().Do(func() {
									close(c)
									c = nil
								}),
							)
							c <- current
							Eventually(isDone).Should(BeTrue())
						},
						Entry("PREPARE", boruta.PREPARE),
						Entry("BUSY", boruta.BUSY),
					)
				})
			})
		})

		Describe("SetGroups", func() {
			It("should fail to SetGroup of nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.SetGroups(uuid, nil)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to SetGroup", func() {
				var group boruta.Groups = []boruta.Group{
					"group1",
				}

				var groups boruta.Groups = []boruta.Group{
					"group1",
					"group2",
					"foobarbaz",
				}

				By("setting it")
				err := wl.SetGroups(worker, group)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers[worker].Groups).To(Equal(group))
				wl.mutex.RUnlock()

				By("setting multiple groups")
				err = wl.SetGroups(worker, groups)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(sort.StringsAreSorted(wl.workers[worker].Groups)).To(BeTrue())
				Expect(len(wl.workers[worker].Groups)).To(Equal(len(groups)))
				for _, g := range groups {
					Expect(wl.workers[worker].Groups).To(ContainElement(g))
				}
				wl.mutex.RUnlock()

				By("setting it to nil")
				err = wl.SetGroups(worker, nil)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers[worker].Groups).To(BeNil())
				wl.mutex.RUnlock()
			})

			Describe("SetGroup with ChangeListener", func() {
				var ctrl *gomock.Controller
				var wc *MockWorkerChange

				BeforeEach(func() {
					ctrl = gomock.NewController(GinkgoT())
					wc = NewMockWorkerChange(ctrl)
					wl.SetChangeListener(wc)
					Expect(wl.changeListener).To(Equal(wc))
				})
				AfterEach(func() {
					ctrl.Finish()
				})

				It("should notify changeListener if set and worker's state is IDLE", func() {
					wl.mutex.RLock()
					wl.workers[worker].State = boruta.IDLE
					wl.mutex.RUnlock()

					wc.EXPECT().OnWorkerIdle(worker)
					err := wl.SetGroups(worker, nil)
					Expect(err).ToNot(HaveOccurred())
				})
				It("should not notify changeListener if set and worker's state is other than IDLE", func() {
					for _, state := range []boruta.WorkerState{boruta.MAINTENANCE, boruta.FAIL, boruta.RUN, boruta.PREPARE, boruta.BUSY} {
						By(string(state))

						wl.mutex.RLock()
						wl.workers[worker].State = state
						wl.mutex.RUnlock()

						err := wl.SetGroups(worker, nil)
						Expect(err).ToNot(HaveOccurred())
					}
				})
			})
			It("should not notify changeListener if not set", func() {
				for _, state := range []boruta.WorkerState{boruta.MAINTENANCE, boruta.FAIL, boruta.RUN, boruta.IDLE, boruta.PREPARE, boruta.BUSY} {
					By(string(state))

					wl.mutex.RLock()
					wl.workers[worker].State = state
					wl.mutex.RUnlock()

					err := wl.SetGroups(worker, nil)
					Expect(err).ToNot(HaveOccurred())
				}
			})
		})

		Describe("ListWorkers", func() {
			var refWorkerList []boruta.WorkerInfo
			var si boruta.SortInfo
			var pag boruta.WorkersPaginator

			getSortedWorkerList := func(l []boruta.WorkerInfo,
				s *boruta.SortInfo) (ret []boruta.WorkerInfo) {

				ret = make([]boruta.WorkerInfo, len(l))
				copy(ret, l)
				sorter, _ := newSorter(s)
				sorter.list = ret
				sort.Sort(sorter)
				return
			}

			indexOf := func(uuid boruta.WorkerUUID,
				w []boruta.WorkerInfo) int {
				for i, v := range w {
					if v.WorkerUUID == uuid {
						return i
					}
				}
				return -1
			}

			registerAndSetGroups := func(groups boruta.Groups, caps boruta.Capabilities) boruta.WorkerInfo {
				capsUUID := getUUID()
				caps[UUID] = capsUUID
				err := wl.Register(caps, dryadAddr.String(), sshdAddr.String())
				Expect(err).ToNot(HaveOccurred())
				workerID := boruta.WorkerUUID(capsUUID)

				err = wl.SetGroups(workerID, groups)
				Expect(err).ToNot(HaveOccurred())

				wl.mutex.RLock()
				info := wl.workers[workerID].WorkerInfo
				wl.mutex.RUnlock()

				return info
			}

			BeforeEach(func() {
				si = boruta.SortInfo{Order: boruta.SortOrderDesc}
				pag = boruta.WorkersPaginator{}
				refWorkerList = make([]boruta.WorkerInfo, 1)
				// Add worker with minimal caps and empty groups.
				wl.mutex.RLock()
				refWorkerList[0] = wl.workers[worker].WorkerInfo
				wl.mutex.RUnlock()
				// Add worker with both groups and caps declared.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					boruta.Groups{"all", "small_1", "small_2"},
					boruta.Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
				// Add worker similar to the second one, but without caps.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					boruta.Groups{"all", "small_1", "small_2"},
					boruta.Capabilities{},
				))
				// Add worker similar to the second one, but without groups.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					boruta.Groups{},
					boruta.Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
				// Add worker similar to the second one, but with display set to no.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					boruta.Groups{"all", "small_1", "small_2"},
					boruta.Capabilities{
						"target":  "yes",
						"display": "no",
					}))
				// Add worker similar to the second one, but absent from small_1 group.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					boruta.Groups{"all", "small_2"},
					boruta.Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
			})

			testWorkerList := func(filter *filter.Workers, si *boruta.SortInfo,
				pag *boruta.WorkersPaginator, present []boruta.WorkerInfo,
				expected *boruta.ListInfo) {

				workers, info, err := wl.ListWorkers(filter, si, pag)
				Expect(err).ToNot(HaveOccurred())
				Expect(workers).To(Equal(present))
				Expect(info).To(Equal(expected))
			}

			It("should return first page of all workers when only SortInfo is set",
				func() {
					testWorkerList(nil, &si, nil,
						getSortedWorkerList(refWorkerList, &si),
						&boruta.ListInfo{
							TotalItems: uint64(len(refWorkerList)),
						})
				})

			It("should return first page of all workers when parameters are empty",
				func() {
					f := filter.NewWorkers(boruta.Groups{}, boruta.Capabilities{})
					testWorkerList(f, &si, &pag,
						getSortedWorkerList(refWorkerList, &si), &boruta.ListInfo{
							TotalItems: uint64(len(refWorkerList)),
						})
				})

			It("should return first page of all workers when parameters are nil",
				func() {
					testWorkerList(nil, nil, nil,
						getSortedWorkerList(refWorkerList, nil), &boruta.ListInfo{
							TotalItems: uint64(len(refWorkerList)),
						})
				})

			It("should return first page of all workers when filter is nil, but interface isn't.",
				func() {
					var f *filter.Workers
					testWorkerList(f, nil, nil,
						getSortedWorkerList(refWorkerList, nil), &boruta.ListInfo{
							TotalItems:     uint64(len(refWorkerList)),
							RemainingItems: 0,
						})
				})

			It("should return first page of all workers when paginator is set with empty ID",
				func() {
					p := &boruta.WorkersPaginator{
						ID:        "",
						Direction: boruta.DirectionForward,
						Limit:     2,
					}
					ref := getSortedWorkerList(refWorkerList, &si)
					testWorkerList(nil, &si, p, ref[:2],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(len(ref) - 2),
						})
				})

			It("should return appropriate page of all workers when forward paginator is set",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[2].WorkerUUID,
						Direction: boruta.DirectionForward,
						Limit:     2,
					}
					testWorkerList(nil, &si, p, ref[3:5],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(len(ref) - 5),
						})
				})

			It("should return appropriate page of filtered workers when forward paginator is set with ID belonging to results",
				func() {
					f := filter.NewWorkers(boruta.Groups{"all"}, nil)
					ref := getSortedWorkerList([]boruta.WorkerInfo{
						refWorkerList[1], refWorkerList[2], refWorkerList[4],
						refWorkerList[5]}, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[1].WorkerUUID,
						Direction: boruta.DirectionForward,
						Limit:     1,
					}
					testWorkerList(f, &si, p, ref[2:3],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(1),
						})
				})

			It("should return appropriate page of filtered workers when backward paginator is set with ID belonging to results",
				func() {
					f := filter.NewWorkers(boruta.Groups{"all"}, nil)
					ref := getSortedWorkerList([]boruta.WorkerInfo{
						refWorkerList[1], refWorkerList[2], refWorkerList[4],
						refWorkerList[5]}, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[2].WorkerUUID,
						Direction: boruta.DirectionBackward,
						Limit:     1,
					}
					testWorkerList(f, &si, p, ref[1:2],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(1),
						})
				})

			It("should return appropriate page of filtered workers when forward paginator is set with ID not belonging to results",
				func() {
					f := filter.NewWorkers(boruta.Groups{"all"}, nil)
					// Varying UUIDs can be painful, in this test case, so use
					// fixed ones.
					wl2 := NewWorkerList()
					refWL := make([]boruta.WorkerInfo, len(refWorkerList))
					for i := 0; i < len(refWorkerList); i++ {
						id := boruta.WorkerUUID(fmt.Sprintf("%d", i))
						wl2.Register(boruta.Capabilities{UUID: string(id)},
							dryadAddr.String(), sshdAddr.String())
						wl2.SetGroups(id, refWorkerList[i].Groups)
						wl2.mutex.RLock()
						refWL[i] = wl2.workers[id].WorkerInfo
						wl2.mutex.RUnlock()
					}
					ref := getSortedWorkerList([]boruta.WorkerInfo{
						refWL[1], refWL[2], refWL[3], refWL[4], refWL[5]},
						&si)

					p := &boruta.WorkersPaginator{
						ID:        refWL[3].WorkerUUID,
						Direction: boruta.DirectionForward,
						Limit:     1,
					}

					i := indexOf(p.ID, ref)
					workers, info, err := wl2.ListWorkers(f, &si, p)
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(Equal(ref[i+1 : i+2]))
					Expect(info).To(Equal(&boruta.ListInfo{
						TotalItems:     uint64(len(ref) - 1),
						RemainingItems: uint64(len(ref) - i - 2),
					}))
				})

			It("should return appropriate page of filtered workers when backward paginator is set with ID not belonging to results",
				func() {
					f := filter.NewWorkers(boruta.Groups{"all"}, nil)
					// Varying UUIDs can be painful, in this test case, so use
					// fixed ones.
					wl2 := NewWorkerList()
					refWL := make([]boruta.WorkerInfo, len(refWorkerList))
					for i := 0; i < len(refWorkerList); i++ {
						id := boruta.WorkerUUID(fmt.Sprintf("%d", i))
						wl2.Register(boruta.Capabilities{UUID: string(id)},
							dryadAddr.String(), sshdAddr.String())
						wl2.SetGroups(id, refWorkerList[i].Groups)
						wl2.mutex.RLock()
						refWL[i] = wl2.workers[id].WorkerInfo
						wl2.mutex.RUnlock()
					}
					ref := getSortedWorkerList([]boruta.WorkerInfo{
						refWL[1], refWL[2], refWL[3], refWL[4], refWL[5]},
						&si)

					p := &boruta.WorkersPaginator{
						ID:        refWL[3].WorkerUUID,
						Direction: boruta.DirectionBackward,
						Limit:     1,
					}

					i := indexOf(p.ID, ref)
					workers, info, err := wl2.ListWorkers(f, &si, p)
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(Equal(ref[i-1 : i]))
					Expect(info).To(Equal(&boruta.ListInfo{
						TotalItems:     uint64(len(ref) - 1),
						RemainingItems: uint64(i - 1),
					}))
				})

			It("should return appropriate page when forward paginator is set with limit greater than numer of workers",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[2].WorkerUUID,
						Direction: boruta.DirectionForward,
						Limit:     uint16(len(ref) + 2),
					}
					testWorkerList(nil, &si, p, ref[3:],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return all workers when forward paginator is set withoud ID and with limit greater than numer of workers",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						Direction: boruta.DirectionForward,
						Limit:     uint16(len(ref) + 2),
					}
					testWorkerList(nil, &si, p, ref,
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return appropriate page when backward paginator is set with limit greater than numer of workers",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[2].WorkerUUID,
						Direction: boruta.DirectionBackward,
						Limit:     uint16(len(ref) + 2),
					}
					testWorkerList(nil, &si, p, ref[:2],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return all workers when backward paginator is set without ID and with limit greater than numer of workers",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						Direction: boruta.DirectionBackward,
						Limit:     uint16(len(ref) + 2),
					}
					testWorkerList(nil, &si, p, ref,
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return appropriate page of all workers when backward paginator is set",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[4].WorkerUUID,
						Direction: boruta.DirectionBackward,
						Limit:     2,
					}
					testWorkerList(nil, &si, p, ref[2:4],
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(2),
						})
				})

			It("should return empty page of all workers when forward paginator is set with ID of last worker",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[len(ref)-1].WorkerUUID,
						Direction: boruta.DirectionForward,
						Limit:     2,
					}
					testWorkerList(nil, &si, p, []boruta.WorkerInfo{},
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return empty page of all workers when backward paginator is set with ID of 1st worker",
				func() {
					ref := getSortedWorkerList(refWorkerList, &si)
					p := &boruta.WorkersPaginator{
						ID:        ref[0].WorkerUUID,
						Direction: boruta.DirectionBackward,
						Limit:     2,
					}
					testWorkerList(nil, &si, p, []boruta.WorkerInfo{},
						&boruta.ListInfo{
							TotalItems:     uint64(len(ref)),
							RemainingItems: uint64(0),
						})
				})

			It("should return first page with MaxPageLimit items when queue has more elements than MaxPageLimit",
				func() {
					if testing.Short() {
						Skip("Big queue - skip when race is on")
					}
					wl2 := NewWorkerList()
					n := int(boruta.MaxPageLimit) + 8
					for i := 0; i < n; i++ {
						wl2.Register(boruta.Capabilities{UUID: getUUID()},
							dryadAddr.String(), sshdAddr.String())
					}
					wl2.mutex.RLock()
					Expect(len(wl2.workers)).To(Equal(n))
					wl2.mutex.RUnlock()
					workers, info, err := wl2.ListWorkers(nil, nil, nil)
					Expect(err).To(BeNil())
					Expect(info.TotalItems).To(Equal(uint64(n)))
					Expect(info.RemainingItems).To(Equal(uint64(n - boruta.MaxPageLimit)))
					Expect(len(workers)).To(Equal(boruta.MaxPageLimit))
				})

			It("should return an error when SortInfo has unknown item", func() {
				sortinfo := &boruta.SortInfo{Item: "foobar"}
				workers, info, err := wl.ListWorkers(nil, sortinfo, nil)
				Expect(len(workers)).To(Equal(0))
				Expect(info).To(Equal((*boruta.ListInfo)(nil)))
				Expect(err).To(Equal(boruta.ErrWrongSortItem))
			})

			It("should return an error when paginator has unknown UUID", func() {
				p := &boruta.WorkersPaginator{ID: "foo"}
				workers, info, err := wl.ListWorkers(nil, nil, p)
				Expect(len(workers)).To(Equal(0))
				Expect(info).To(Equal((*boruta.ListInfo)(nil)))
				Expect(err).To(Equal(boruta.NotFoundError("worker")))
			})

			Describe("filterCaps", func() {
				It("should return first page of all workers satisfying defined caps",
					func() {
						By("Returning first page of all workers with display")
						ref := getSortedWorkerList([]boruta.WorkerInfo{
							refWorkerList[1], refWorkerList[3],
							refWorkerList[5]}, &si)
						testWorkerList(filter.NewWorkers(boruta.Groups{},
							boruta.Capabilities{"display": "yes"}), &si,
							&pag, ref,
							&boruta.ListInfo{TotalItems: 3})

						By("Returning first page of all workers without display")
						testWorkerList(filter.NewWorkers(boruta.Groups{},
							boruta.Capabilities{"display": "no"}), &si, &pag,
							[]boruta.WorkerInfo{refWorkerList[4]},
							&boruta.ListInfo{TotalItems: 1})
					})

				It("should return empty list if no worker matches the caps", func() {
					workers, info, err := wl.ListWorkers(filter.NewWorkers(boruta.Groups{},
						boruta.Capabilities{
							"non-existing-caps": "",
						}), &si, &pag)
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(BeEmpty())
					Expect(info).To(Equal(&boruta.ListInfo{}))
				})
			})

			Describe("filterGroups", func() {
				It("should return first page of all workers satisfying defined groups",
					func() {
						By("Returning first page of all workers in group all")
						f := filter.NewWorkers(boruta.Groups{"all"}, nil)
						ref := []boruta.WorkerInfo{refWorkerList[1],
							refWorkerList[2], refWorkerList[4],
							refWorkerList[5]}
						testWorkerList(f, &si, &pag,
							getSortedWorkerList(ref, &si),
							&boruta.ListInfo{TotalItems: 4})

						By("Returning first page of all workers in group small_1")
						f = filter.NewWorkers(boruta.Groups{"small_1"}, nil)
						ref = []boruta.WorkerInfo{refWorkerList[1],
							refWorkerList[2], refWorkerList[4]}
						testWorkerList(f, &si, &pag,
							getSortedWorkerList(ref, &si),
							&boruta.ListInfo{TotalItems: 3})
					})

				It("should return empty list if no worker matches the group", func() {
					workers, info, err := wl.ListWorkers(filter.NewWorkers(boruta.Groups{"non-existing-group"},
						nil), &si, &pag)
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(BeEmpty())
					Expect(info).To(Equal(&boruta.ListInfo{}))
				})
			})

			It("should work with many groups and caps defined", func() {
				By("Returning first page of all targets with display in both groups")
				ref := getSortedWorkerList([]boruta.WorkerInfo{refWorkerList[1],
					refWorkerList[5]}, &si)
				testWorkerList(filter.NewWorkers(boruta.Groups{"small_1", "small_2"},
					boruta.Capabilities{
						"target":  "yes",
						"display": "yes",
					}), &si, &pag, ref, &boruta.ListInfo{TotalItems: 2})

				By("Returning first page of all targets without display in group all and small_1")
				ref = getSortedWorkerList([]boruta.WorkerInfo{refWorkerList[4]}, &si)
				testWorkerList(filter.NewWorkers(boruta.Groups{"all", "small_1"},
					boruta.Capabilities{
						"target":  "yes",
						"display": "no",
					}), &si, &pag, ref, &boruta.ListInfo{TotalItems: 1})
			})
		})

		Describe("GetWorkerInfo", func() {
			It("should fail to GetWorkerInfo of nonexistent worker", func() {
				uuid := randomUUID()
				_, err := wl.GetWorkerInfo(uuid)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to GetWorkerInfo", func() {
				workerInfo, err := wl.GetWorkerInfo(worker)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(workerInfo).To(Equal(wl.workers[worker].WorkerInfo))
				wl.mutex.RUnlock()
			})
		})

		Describe("Setters and Getters", func() {
			type genericGet func(wl *WorkerList, uuid boruta.WorkerUUID, expectedItem interface{}, expectedErr error)
			getDryad := genericGet(func(wl *WorkerList, uuid boruta.WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.getWorkerAddr(uuid)
				if expectedErr != nil {
					Expect(item).To(Equal(net.TCPAddr{}))
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(item).To(Equal(expectedItem.(net.TCPAddr)))
			})
			getSSH := genericGet(func(wl *WorkerList, uuid boruta.WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.GetWorkerSSHAddr(uuid)
				if expectedErr != nil {
					Expect(item).To(Equal(net.TCPAddr{}))
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(item).To(Equal(expectedItem.(net.TCPAddr)))
			})
			getKey := genericGet(func(wl *WorkerList, uuid boruta.WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.GetWorkerKey(uuid)
				if expectedErr != nil {
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(&item).To(Equal(expectedItem.(*rsa.PrivateKey)))
			})
			getters := []genericGet{getKey, getDryad, getSSH}

			type genericSet func(wl *WorkerList, uuid boruta.WorkerUUID, expectedErr error) interface{}
			setKey := genericSet(func(wl *WorkerList, uuid boruta.WorkerUUID, expectedErr error) interface{} {
				key, err := rsa.GenerateKey(rand.Reader, 128)
				Expect(err).ToNot(HaveOccurred())
				err = wl.setWorkerKey(uuid, key)
				if expectedErr != nil {
					Expect(err).To(Equal(expectedErr))
					return nil
				}
				Expect(err).ToNot(HaveOccurred())
				return key
			})
			setters := []genericSet{setKey}

			It("should fail to get information of nonexistent worker", func() {
				uuid := randomUUID()
				for _, fn := range getters {
					fn(wl, uuid, nil, ErrWorkerNotFound)
				}
			})

			It("should fail to set information of nonexistent worker", func() {
				uuid := randomUUID()
				for _, fn := range setters {
					fn(wl, uuid, ErrWorkerNotFound)
				}
			})

			It("should work to set and get information", func() {
				// There's only 1 setter and 3 getters, so only 1st getter is checked in loop.
				for i, set := range setters {
					get := getters[i]
					get(wl, worker, set(wl, worker, nil), nil)
				}
				getDryad(wl, worker, *dryadAddr, nil)
				getSSH(wl, worker, *sshdAddr, nil)
			})
		})
		Describe("PrepareWorker", func() {
			var ctrl *gomock.Controller
			var dcm *MockDryadClientManager
			ip := net.IPv4(2, 4, 6, 8)
			testerr := errors.New("Test Error")
			noWorker := boruta.WorkerUUID("There's no such worker")

			eventuallyKey := func(info *mapWorker, match types.GomegaMatcher) {
				EventuallyWithOffset(1, func() *rsa.PrivateKey {
					wl.mutex.RLock()
					defer wl.mutex.RUnlock()
					return info.key
				}).Should(match)
			}
			eventuallyState := func(info *mapWorker, state boruta.WorkerState) {
				EventuallyWithOffset(1, func() boruta.WorkerState {
					wl.mutex.RLock()
					defer wl.mutex.RUnlock()
					return info.State
				}).Should(Equal(state))
			}

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				dcm = NewMockDryadClientManager(ctrl)
				wl.newDryadClient = func() dryad.ClientManager {
					return dcm
				}
			})
			AfterEach(func() {
				ctrl.Finish()
			})

			It("should set worker in RUN state into IDLE in without-key preparation", func() {
				wl.mutex.RLock()
				wl.workers[worker].State = boruta.RUN
				wl.mutex.RUnlock()
				err := wl.PrepareWorker(worker, false)
				Expect(err).NotTo(HaveOccurred())
				wl.mutex.RLock()
				info, ok := wl.workers[worker]
				wl.mutex.RUnlock()
				Expect(ok).To(BeTrue())
				Expect(info.State).To(Equal(boruta.IDLE))
			})

			DescribeTable("should fail to set worker into IDLE in without-key preparation",
				func(from boruta.WorkerState) {
					wl.mutex.RLock()
					wl.workers[worker].State = from
					wl.mutex.RUnlock()
					err := wl.PrepareWorker(worker, false)
					Expect(err).To(Equal(ErrForbiddenStateChange))
					wl.mutex.RLock()
					info, ok := wl.workers[worker]
					wl.mutex.RUnlock()
					Expect(ok).To(BeTrue())
					Expect(info.State).To(Equal(from))
				},
				Entry("MAINTENANCE", boruta.MAINTENANCE),
				Entry("IDLE", boruta.IDLE),
				Entry("FAIL", boruta.FAIL),
				Entry("PREPARE", boruta.PREPARE),
				Entry("BUSY", boruta.BUSY),
			)

			It("should fail to prepare not existing worker in without-key preparation", func() {
				err := wl.PrepareWorker(noWorker, false)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should fail to prepare not existing worker in with-key preparation", func() {
				err := wl.PrepareWorker(noWorker, true)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			Describe("with worker's IP set", func() {
				var info *mapWorker

				BeforeEach(func() {
					var ok bool
					info, ok = wl.workers[worker]
					Expect(ok).To(BeTrue())
					Expect(info.key).To(BeNil())
					info.dryad = new(net.TCPAddr)
					info.dryad.IP = ip
					info.State = boruta.RUN
				})

				It("should set worker into IDLE state and prepare a key", func() {
					gomock.InOrder(
						dcm.EXPECT().Create(info.dryad),
						dcm.EXPECT().Prepare(gomock.Any()).Return(nil),
						dcm.EXPECT().Close().Do(func() {
							wl.mutex.Lock()
							Expect(info.State).To(Equal(boruta.PREPARE))
							wl.mutex.Unlock()
						}),
					)

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, boruta.IDLE)
					eventuallyKey(info, Not(Equal(&rsa.PrivateKey{})))
				})

				It("should fail to prepare worker if dryadClientManager fails to prepare client", func() {
					gomock.InOrder(
						dcm.EXPECT().Create(info.dryad),
						dcm.EXPECT().Prepare(gomock.Any()).Return(testerr),
						dcm.EXPECT().Close().Do(func() {
							wl.mutex.Lock()
							Expect(info.State).To(Equal(boruta.PREPARE))
							wl.mutex.Unlock()
						}),
					)

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, boruta.FAIL)
					Expect(info.key).To(BeNil())
				})

				It("should fail to prepare worker if dryadClientManager fails to create client", func() {
					dcm.EXPECT().Create(info.dryad).DoAndReturn(func(*net.TCPAddr) error {
						wl.mutex.Lock()
						Expect(info.State).To(Equal(boruta.PREPARE))
						wl.mutex.Unlock()
						return testerr
					})

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, boruta.FAIL)
					Expect(info.key).To(BeNil())
				})

				It("should fail to prepare worker if key generation fails", func() {
					gomock.InOrder(
						dcm.EXPECT().Create(info.dryad),
						dcm.EXPECT().Close().Do(func() {
							wl.mutex.Lock()
							Expect(info.State).To(Equal(boruta.PREPARE))
							wl.mutex.Unlock()
						}),
					)

					sizeRSA = 1
					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, boruta.FAIL)
					Expect(info.key).To(BeNil())
				})
			})
		})

		Describe("setState with changeListener", func() {
			var ctrl *gomock.Controller
			var wc *MockWorkerChange

			set := func(state boruta.WorkerState) {
				wl.mutex.Lock()
				wl.workers[worker].State = state
				wl.mutex.Unlock()
			}
			check := func(state boruta.WorkerState) {
				wl.mutex.RLock()
				Expect(wl.workers[worker].State).To(Equal(state))
				wl.mutex.RUnlock()
			}
			swap := func(newChan chan boruta.WorkerState) (oldChan chan boruta.WorkerState) {
				wl.mutex.Lock()
				oldChan = wl.workers[worker].backgroundOperation
				wl.workers[worker].backgroundOperation = newChan
				wl.mutex.Unlock()
				return
			}
			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				wc = NewMockWorkerChange(ctrl)
				wl.SetChangeListener(wc)
				Expect(wl.changeListener).To(Equal(wc))
			})
			AfterEach(func() {
				ctrl.Finish()
			})
			DescribeTable("Should change state without calling changeListener",
				func(from, to boruta.WorkerState) {
					set(from)
					newChan := make(chan boruta.WorkerState, backgroundOperationsBufferSize)
					oldChan := swap(newChan)
					defer swap(oldChan)

					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
					Eventually(newChan).Should(Receive(Equal(to)))
				},
				Entry("MAINTENANCE->MAINTENANCE", boruta.MAINTENANCE, boruta.MAINTENANCE),
				Entry("MAINTENANCE->RUN", boruta.MAINTENANCE, boruta.RUN),
				Entry("MAINTENANCE->FAIL", boruta.MAINTENANCE, boruta.FAIL),
				Entry("MAINTENANCE->PREPARE", boruta.MAINTENANCE, boruta.PREPARE),
				Entry("MAINTENANCE->BUSY", boruta.MAINTENANCE, boruta.BUSY),
				Entry("IDLE->MAINTENANCE", boruta.IDLE, boruta.MAINTENANCE),
				Entry("IDLE->RUN", boruta.IDLE, boruta.RUN),
				Entry("IDLE->FAIL", boruta.IDLE, boruta.FAIL),
				Entry("IDLE->PREPARE", boruta.IDLE, boruta.PREPARE),
				Entry("IDLE->BUSY", boruta.IDLE, boruta.BUSY),
				Entry("RUN->PREPARE", boruta.RUN, boruta.PREPARE),
				Entry("FAIL->MAINTENANCE", boruta.FAIL, boruta.MAINTENANCE),
				Entry("FAIL->RUN", boruta.FAIL, boruta.RUN),
				Entry("FAIL->FAIL", boruta.FAIL, boruta.FAIL),
				Entry("FAIL->PREPARE", boruta.FAIL, boruta.PREPARE),
				Entry("FAIL->BUSY", boruta.FAIL, boruta.BUSY),
				Entry("PREPARE->MAINTENANCE", boruta.PREPARE, boruta.MAINTENANCE),
				Entry("PREPARE->RUN", boruta.PREPARE, boruta.RUN),
				Entry("PREPARE->FAIL", boruta.PREPARE, boruta.FAIL),
				Entry("PREPARE->PREPARE", boruta.PREPARE, boruta.PREPARE),
				Entry("PREPARE->BUSY", boruta.PREPARE, boruta.BUSY),
				Entry("BUSY->MAINTENANCE", boruta.BUSY, boruta.MAINTENANCE),
				Entry("BUSY->RUN", boruta.BUSY, boruta.RUN),
				Entry("BUSY->FAIL", boruta.BUSY, boruta.FAIL),
				Entry("BUSY->PREPARE", boruta.BUSY, boruta.PREPARE),
				Entry("BUSY->BUSY", boruta.BUSY, boruta.BUSY),
			)

			DescribeTable("Should change state and call OnWorkerIdle",
				func(from, to boruta.WorkerState) {
					set(from)
					newChan := make(chan boruta.WorkerState, backgroundOperationsBufferSize)
					oldChan := swap(newChan)
					defer swap(oldChan)
					wc.EXPECT().OnWorkerIdle(worker)

					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
					Eventually(newChan).Should(Receive(Equal(to)))
				},
				Entry("MAINTENANCE->IDLE", boruta.MAINTENANCE, boruta.IDLE),
				Entry("IDLE->IDLE", boruta.IDLE, boruta.IDLE),
				Entry("RUN->IDLE", boruta.RUN, boruta.IDLE),
				Entry("FAIL->IDLE", boruta.FAIL, boruta.IDLE),
				Entry("PREPARE->IDLE", boruta.PREPARE, boruta.IDLE),
				Entry("BUSY->IDLE", boruta.BUSY, boruta.IDLE),
			)

			DescribeTable("Should change state and call OnWorkerFail",
				func(from, to boruta.WorkerState) {
					set(from)
					newChan := make(chan boruta.WorkerState, backgroundOperationsBufferSize)
					oldChan := swap(newChan)
					defer swap(oldChan)
					wc.EXPECT().OnWorkerFail(worker)

					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
					Eventually(newChan).Should(Receive(Equal(to)))
				},
				Entry("RUN->MAINTENANCE", boruta.RUN, boruta.MAINTENANCE),
				Entry("RUN->RUN", boruta.RUN, boruta.RUN),
				Entry("RUN->FAIL", boruta.RUN, boruta.FAIL),
				Entry("RUN->BUSY", boruta.RUN, boruta.BUSY),
			)
		})
	})

	Describe("TakeBestMatchingWorker", func() {
		addWorker := func(groups boruta.Groups, caps boruta.Capabilities) *mapWorker {
			capsUUID := getUUID()
			workerUUID := boruta.WorkerUUID(capsUUID)

			caps[UUID] = capsUUID
			wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			wl.mutex.RLock()
			w, ok := wl.workers[workerUUID]
			wl.mutex.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(w.State).To(Equal(boruta.MAINTENANCE))

			err := wl.SetGroups(workerUUID, groups)
			Expect(err).NotTo(HaveOccurred())

			return w
		}
		addIdleWorker := func(groups boruta.Groups, caps boruta.Capabilities) *mapWorker {
			w := addWorker(groups, caps)
			w.State = boruta.IDLE
			return w
		}
		generateGroups := func(count int) boruta.Groups {
			var groups boruta.Groups
			for i := 0; i < count; i++ {
				groups = append(groups, boruta.Group(fmt.Sprintf("testGroup_%d", i)))
			}
			return groups
		}
		generateCaps := func(count int) boruta.Capabilities {
			caps := make(boruta.Capabilities)
			for i := 0; i < count; i++ {
				k := fmt.Sprintf("testCapKey_%d", i)
				v := fmt.Sprintf("testCapValue_%d", i)
				caps[k] = v
			}
			return caps
		}

		It("should fail to find matching worker when there are no workers", func() {
			ret, err := wl.TakeBestMatchingWorker(boruta.Groups{}, boruta.Capabilities{})
			Expect(err).To(Equal(ErrNoMatchingWorker))
			Expect(ret).To(BeZero())
		})

		It("should match fitting worker and set it into RUN state", func() {
			w := addIdleWorker(boruta.Groups{}, boruta.Capabilities{})

			ret, err := wl.TakeBestMatchingWorker(boruta.Groups{}, boruta.Capabilities{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(Equal(w.WorkerUUID))
			Expect(w.State).To(Equal(boruta.RUN))
		})

		It("should not match not IDLE workers", func() {
			addWorker(boruta.Groups{}, boruta.Capabilities{})

			ret, err := wl.TakeBestMatchingWorker(boruta.Groups{}, boruta.Capabilities{})
			Expect(err).To(Equal(ErrNoMatchingWorker))
			Expect(ret).To(BeZero())
		})

		It("should choose least capable worker", func() {
			// Create matching workers.
			w5g5c := addIdleWorker(generateGroups(5), generateCaps(5))
			w1g7c := addIdleWorker(generateGroups(1), generateCaps(7))
			w5g1c := addIdleWorker(generateGroups(5), generateCaps(1))
			// Create non-matching workers.
			w2g0c := addIdleWorker(generateGroups(2), generateCaps(0))
			w0g2c := addIdleWorker(generateGroups(0), generateCaps(2))

			expectedWorkers := []*mapWorker{w5g1c, w1g7c, w5g5c}
			for _, w := range expectedWorkers {
				ret, err := wl.TakeBestMatchingWorker(generateGroups(1), generateCaps(1))
				Expect(err).NotTo(HaveOccurred())
				Expect(ret).To(Equal(w.WorkerUUID))
				Expect(w.State).To(Equal(boruta.RUN))
			}
			ret, err := wl.TakeBestMatchingWorker(generateGroups(1), generateCaps(1))
			Expect(err).To(Equal(ErrNoMatchingWorker))
			Expect(ret).To(BeZero())

			leftWorkers := []*mapWorker{w2g0c, w0g2c}
			for _, w := range leftWorkers {
				Expect(w.State).To(Equal(boruta.IDLE))
			}
		})
	})

	Describe("SetChangeListener", func() {
		It("should set WorkerChange", func() {
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
			wc := NewMockWorkerChange(ctrl)

			Expect(wl.changeListener).To(BeNil())
			wl.SetChangeListener(wc)
			Expect(wl.changeListener).To(Equal(wc))
		})
	})
})
