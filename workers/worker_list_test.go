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

//go:generate mockgen -package workers -destination=dryadclientmanager_mock_test.go -write_package_comment=false -mock_names ClientManager=MockDryadClientManager git.tizen.org/tools/boruta/rpc/dryad ClientManager

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/rpc/dryad"

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
				_, ok := wl.workers[WorkerUUID(uuid)]
				Expect(ok).To(BeTrue())
			}
			// Check if all workers from the wl.workers are present
			for _, workerInfo := range wl.workers {
				ok := false
				for _, uuid := range registeredWorkers {
					if workerInfo.WorkerUUID == WorkerUUID(uuid) {
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

		getRandomCaps := func() Capabilities {
			return Capabilities{
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
			uuid := WorkerUUID(caps[UUID])
			wl.mutex.RLock()
			defer wl.mutex.RUnlock()
			Expect(wl.workers).To(HaveKey(uuid))
			Expect(wl.workers[uuid].State).To(Equal(MAINTENANCE))
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
			Expect(wl.workers[WorkerUUID(caps[UUID])].Caps).To(Equal(caps))
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
		var worker WorkerUUID

		randomUUID := func() WorkerUUID {
			newUUID := worker
			for newUUID == worker {
				newUUID = WorkerUUID(getUUID())
			}
			return newUUID
		}
		registerWorker := func() WorkerUUID {
			capsUUID := randomUUID()
			err := wl.Register(Capabilities{UUID: string(capsUUID)}, dryadAddr.String(), sshdAddr.String())
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

		Describe("SetFail", func() {
			It("should fail to SetFail of nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.SetFail(uuid, "")
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to SetFail", func() {
				for _, state := range []WorkerState{IDLE, RUN} {
					wl.mutex.Lock()
					wl.workers[worker].State = state
					wl.mutex.Unlock()
					err := wl.SetFail(worker, "")
					Expect(err).ToNot(HaveOccurred())
					wl.mutex.RLock()
					Expect(wl.workers[worker].State).To(Equal(FAIL))
					wl.mutex.RUnlock()
				}
			})

			It("Should fail to SetFail in MAINTENANCE state", func() {
				wl.mutex.Lock()
				Expect(wl.workers[worker].State).To(Equal(MAINTENANCE))
				wl.mutex.Unlock()
				err := wl.SetFail(worker, "")
				Expect(err).To(Equal(ErrInMaintenance))
				wl.mutex.RLock()
				Expect(wl.workers[worker].State).To(Equal(MAINTENANCE))
				wl.mutex.RUnlock()
			})
		})

		Describe("Deregister", func() {
			It("should fail to deregister nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.Deregister(uuid)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to deregister", func() {
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

			It("should fail to deregister worker not in MAINTENANCE state", func() {
				for _, state := range []WorkerState{IDLE, RUN, FAIL} {
					wl.mutex.Lock()
					wl.workers[worker].State = state
					wl.mutex.Unlock()
					err := wl.Deregister(worker)
					Expect(err).To(Equal(ErrNotInMaintenance))
					wl.mutex.RLock()
					Expect(wl.workers).To(HaveKey(worker))
					wl.mutex.RUnlock()
				}
			})
		})

		Describe("SetState", func() {
			It("should fail to SetState of nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.SetState(uuid, MAINTENANCE)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should fail to SetState for invalid transitions", func() {
				invalidTransitions := [][]WorkerState{
					{RUN, IDLE},
					{FAIL, IDLE},
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

			It("should fail to SetState for incorrect state argument", func() {
				invalidArgument := [][]WorkerState{
					{MAINTENANCE, RUN},
					{MAINTENANCE, FAIL},
					{IDLE, FAIL},
					{IDLE, RUN},
					{RUN, FAIL},
					{FAIL, RUN},
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
			Describe("with dryadClientManager mockup", func() {
				var ctrl *gomock.Controller
				var dcm *MockDryadClientManager
				ip := net.IPv4(2, 4, 6, 8)
				testerr := errors.New("Test Error")
				var info *mapWorker
				noWorker := WorkerUUID("There's no such worker")
				putStr := "maintenance"

				eventuallyState := func(info *mapWorker, state WorkerState) {
					EventuallyWithOffset(1, func() WorkerState {
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

				Describe("from MAINTENANCE to IDLE", func() {
					BeforeEach(func() {
						wl.mutex.Lock()
						info.State = MAINTENANCE
						wl.mutex.Unlock()
					})

					It("should work to SetState", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.AssignableToTypeOf(&rsa.PublicKey{})).Return(nil),
							dcm.EXPECT().Close(),
						)

						err := wl.SetState(worker, IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, IDLE)
						eventuallyKey(info, Not(Equal(&rsa.PrivateKey{})))
					})

					It("should fail to SetState if dryadClientManager fails to prepare client", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().Prepare(gomock.AssignableToTypeOf(&rsa.PublicKey{})).Return(testerr),
							dcm.EXPECT().Close(),
						)

						err := wl.SetState(worker, IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, FAIL)
						Expect(info.key).To(BeNil())
					})

					It("should fail to SetState if dryadClientManager fails to create client", func() {
						dcm.EXPECT().Create(info.dryad).Return(testerr)

						err := wl.SetState(worker, IDLE)
						Expect(err).ToNot(HaveOccurred())
						eventuallyState(info, FAIL)
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

				fromStates := []WorkerState{IDLE, RUN, FAIL}
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
								dcm.EXPECT().Close(),
							)

							err := wl.SetState(worker, MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyState(info, MAINTENANCE)
						})

						It("should fail to SetState if dryadClientManager fails to put dryad in maintenance state", func() {
							gomock.InOrder(
								dcm.EXPECT().Create(info.dryad),
								dcm.EXPECT().PutInMaintenance(putStr).Return(testerr),
								dcm.EXPECT().Close().Do(func() {
									wl.mutex.Lock()
									info.State = WorkerState("TEST")
									wl.mutex.Unlock()
									setTrigger(1)
								}),
							)

							err := wl.SetState(worker, MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyTrigger(1)
							eventuallyState(info, FAIL)
						})

						It("should fail to SetState if dryadClientManager fails to create client", func() {
							dcm.EXPECT().Create(info.dryad).Return(testerr).Do(func(*net.TCPAddr) {
								wl.mutex.Lock()
								info.State = WorkerState("TEST")
								wl.mutex.Unlock()
								setTrigger(2)
							})

							err := wl.SetState(worker, MAINTENANCE)
							Expect(err).ToNot(HaveOccurred())
							eventuallyTrigger(2)
							eventuallyState(info, FAIL)
						})
					})
				}
				Describe("putInMaintenance", func() {
					It("should work", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr),
							dcm.EXPECT().Close(),
						)

						err := wl.putInMaintenance(worker)
						Expect(err).ToNot(HaveOccurred())
					})

					It("should fail if dryadClientManager fails to put dryad in maintenance state", func() {
						gomock.InOrder(
							dcm.EXPECT().Create(info.dryad),
							dcm.EXPECT().PutInMaintenance(putStr).Return(testerr),
							dcm.EXPECT().Close(),
						)

						err := wl.putInMaintenance(worker)
						Expect(err).To(Equal(testerr))
					})

					It("should fail if dryadClientManager fails to create client", func() {
						dcm.EXPECT().Create(info.dryad).Return(testerr)

						err := wl.putInMaintenance(worker)
						Expect(err).To(Equal(testerr))
					})

					It("should fail if worker is not registered", func() {
						err := wl.putInMaintenance(noWorker)
						Expect(err).To(Equal(ErrWorkerNotFound))
					})
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
				var group Groups = []Group{
					"group1",
				}

				By("setting it")
				err := wl.SetGroups(worker, group)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers[worker].Groups).To(Equal(group))
				wl.mutex.RUnlock()

				By("setting it to nil")
				err = wl.SetGroups(worker, nil)
				Expect(err).ToNot(HaveOccurred())
				wl.mutex.RLock()
				Expect(wl.workers[worker].Groups).To(BeNil())
				wl.mutex.RUnlock()
			})
		})

		Describe("ListWorkers", func() {
			var refWorkerList []WorkerInfo

			registerAndSetGroups := func(groups Groups, caps Capabilities) WorkerInfo {
				capsUUID := getUUID()
				caps[UUID] = capsUUID
				err := wl.Register(caps, dryadAddr.String(), sshdAddr.String())
				Expect(err).ToNot(HaveOccurred())
				workerID := WorkerUUID(capsUUID)

				err = wl.SetGroups(workerID, groups)
				Expect(err).ToNot(HaveOccurred())

				wl.mutex.RLock()
				info := wl.workers[workerID].WorkerInfo
				wl.mutex.RUnlock()

				return info
			}

			BeforeEach(func() {
				refWorkerList = make([]WorkerInfo, 1)
				// Add worker with minimal caps and empty groups.
				wl.mutex.RLock()
				refWorkerList[0] = wl.workers[worker].WorkerInfo
				wl.mutex.RUnlock()
				// Add worker with both groups and caps declared.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					Groups{"all", "small_1", "small_2"},
					Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
				// Add worker similar to the second one, but without caps.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					Groups{"all", "small_1", "small_2"},
					Capabilities{},
				))
				// Add worker similar to the second one, but without groups.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					Groups{},
					Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
				// Add worker similar to the second one, but with display set to no.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					Groups{"all", "small_1", "small_2"},
					Capabilities{
						"target":  "yes",
						"display": "no",
					}))
				// Add worker similar to the second one, but absent from small_1 group.
				refWorkerList = append(refWorkerList, registerAndSetGroups(
					Groups{"all", "small_2"},
					Capabilities{
						"target":  "yes",
						"display": "yes",
					}))
			})

			testWorkerList := func(groups Groups, caps Capabilities,
				present, absent []WorkerInfo) {
				workers, err := wl.ListWorkers(groups, caps)
				Expect(err).ToNot(HaveOccurred())
				for _, workerInfo := range present {
					Expect(workers).To(ContainElement(workerInfo))
				}
				for _, workerInfo := range absent {
					Expect(workers).ToNot(ContainElement(workerInfo))
				}
			}

			It("should return all workers when parameters are nil", func() {
				testWorkerList(nil, nil, refWorkerList, nil)
			})

			It("should return all workers when parameters are empty", func() {
				testWorkerList(Groups{}, Capabilities{}, refWorkerList, nil)
			})

			Describe("filterCaps", func() {
				It("should return all workers satisfying defined caps", func() {
					By("Returning all workers with display")
					testWorkerList(Groups{},
						Capabilities{"display": "yes"},
						[]WorkerInfo{refWorkerList[1], refWorkerList[3], refWorkerList[5]},
						[]WorkerInfo{refWorkerList[0], refWorkerList[2], refWorkerList[4]})

					By("Returning all workers without display")
					testWorkerList(Groups{},
						Capabilities{"display": "no"},
						[]WorkerInfo{refWorkerList[4]},
						[]WorkerInfo{refWorkerList[0], refWorkerList[1],
							refWorkerList[2], refWorkerList[3], refWorkerList[5]})
				})

				It("should return empty list if no worker matches the caps", func() {
					workers, err := wl.ListWorkers(Groups{},
						Capabilities{
							"non-existing-caps": "",
						})
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(BeEmpty())
				})
			})

			Describe("filterGroups", func() {
				It("should return all workers satisfying defined groups", func() {
					By("Returning all workers in group all")
					testWorkerList(Groups{"all"},
						nil,
						[]WorkerInfo{refWorkerList[1], refWorkerList[2],
							refWorkerList[4], refWorkerList[5]},
						[]WorkerInfo{refWorkerList[0], refWorkerList[3]})

					By("Returning all workers in group small_1")
					testWorkerList(Groups{"small_1"},
						nil,
						[]WorkerInfo{refWorkerList[1], refWorkerList[2], refWorkerList[4]},
						[]WorkerInfo{refWorkerList[0], refWorkerList[3], refWorkerList[5]})
				})

				It("should return empty list if no worker matches the group", func() {
					workers, err := wl.ListWorkers(Groups{"non-existing-group"}, nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(workers).To(BeEmpty())
				})
			})

			It("should work with many groups and caps defined", func() {
				By("Returning all targets with display in both groups")
				testWorkerList(Groups{"small_1", "small_2"},
					Capabilities{
						"target":  "yes",
						"display": "yes",
					},
					[]WorkerInfo{refWorkerList[1], refWorkerList[5]},
					[]WorkerInfo{refWorkerList[0], refWorkerList[2],
						refWorkerList[3], refWorkerList[4]})

				By("Returning all targets without display in group all and small_1")
				testWorkerList(Groups{"all", "small_1"},
					Capabilities{
						"target":  "yes",
						"display": "no",
					},
					[]WorkerInfo{refWorkerList[4]},
					[]WorkerInfo{refWorkerList[0], refWorkerList[1],
						refWorkerList[2], refWorkerList[3], refWorkerList[5]})
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
			type genericGet func(wl *WorkerList, uuid WorkerUUID, expectedItem interface{}, expectedErr error)
			getDryad := genericGet(func(wl *WorkerList, uuid WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.GetWorkerAddr(uuid)
				if expectedErr != nil {
					Expect(item).To(Equal(net.TCPAddr{}))
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(item).To(Equal(expectedItem.(net.TCPAddr)))
			})
			getSSH := genericGet(func(wl *WorkerList, uuid WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.GetWorkerSSHAddr(uuid)
				if expectedErr != nil {
					Expect(item).To(Equal(net.TCPAddr{}))
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(item).To(Equal(expectedItem.(net.TCPAddr)))
			})
			getKey := genericGet(func(wl *WorkerList, uuid WorkerUUID, expectedItem interface{}, expectedErr error) {
				item, err := wl.GetWorkerKey(uuid)
				if expectedErr != nil {
					Expect(err).To(Equal(expectedErr))
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(&item).To(Equal(expectedItem.(*rsa.PrivateKey)))
			})
			getters := []genericGet{getKey, getDryad, getSSH}

			type genericSet func(wl *WorkerList, uuid WorkerUUID, expectedErr error) interface{}
			setKey := genericSet(func(wl *WorkerList, uuid WorkerUUID, expectedErr error) interface{} {
				key, err := rsa.GenerateKey(rand.Reader, 128)
				Expect(err).ToNot(HaveOccurred())
				err = wl.SetWorkerKey(uuid, key)
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
				for i, set := range setters {
					get := getters[i]
					get(wl, worker, set(wl, worker, nil), nil)
				}
			})
		})
		Describe("PrepareWorker", func() {
			var ctrl *gomock.Controller
			var dcm *MockDryadClientManager
			ip := net.IPv4(2, 4, 6, 8)
			testerr := errors.New("Test Error")
			noWorker := WorkerUUID("There's no such worker")

			eventuallyKey := func(info *mapWorker, match types.GomegaMatcher) {
				EventuallyWithOffset(1, func() *rsa.PrivateKey {
					wl.mutex.RLock()
					defer wl.mutex.RUnlock()
					return info.key
				}).Should(match)
			}
			eventuallyState := func(info *mapWorker, state WorkerState) {
				EventuallyWithOffset(1, func() WorkerState {
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

			It("should set worker into IDLE in without-key preparation", func() {
				err := wl.PrepareWorker(worker, false)
				Expect(err).NotTo(HaveOccurred())
				wl.mutex.RLock()
				info, ok := wl.workers[worker]
				wl.mutex.RUnlock()
				Expect(ok).To(BeTrue())
				Expect(info.State).To(Equal(IDLE))
			})
			It("should fail to prepare not existing worker in without-key preparation", func() {
				uuid := randomUUID()
				err := wl.PrepareWorker(uuid, false)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})
			It("should ignore to prepare worker for non-existing worker", func() {
				err := wl.PrepareWorker(noWorker, true)
				Expect(err).NotTo(HaveOccurred())
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
				})
				It("should set worker into IDLE state and prepare a key", func() {
					gomock.InOrder(
						dcm.EXPECT().Create(info.dryad),
						dcm.EXPECT().Prepare(gomock.AssignableToTypeOf(&rsa.PublicKey{})).Return(nil),
						dcm.EXPECT().Close(),
					)

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, IDLE)
					eventuallyKey(info, Not(Equal(&rsa.PrivateKey{})))
				})
				It("should fail to prepare worker if dryadClientManager fails to prepare client", func() {
					gomock.InOrder(
						dcm.EXPECT().Create(info.dryad),
						dcm.EXPECT().Prepare(gomock.AssignableToTypeOf(&rsa.PublicKey{})).Return(testerr),
						dcm.EXPECT().Close(),
					)

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, FAIL)
					Expect(info.key).To(BeNil())
				})
				It("should fail to prepare worker if dryadClientManager fails to create client", func() {
					dcm.EXPECT().Create(info.dryad).Return(testerr)

					err := wl.PrepareWorker(worker, true)
					Expect(err).NotTo(HaveOccurred())

					eventuallyState(info, FAIL)
					Expect(info.key).To(BeNil())
				})
			})
		})

		Describe("setState with changeListener", func() {
			var ctrl *gomock.Controller
			var wc *MockWorkerChange

			set := func(state WorkerState) {
				wl.mutex.Lock()
				wl.workers[worker].State = state
				wl.mutex.Unlock()
			}
			check := func(state WorkerState) {
				wl.mutex.RLock()
				Expect(wl.workers[worker].State).To(Equal(state))
				wl.mutex.RUnlock()
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
				func(from, to WorkerState) {
					set(from)
					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
				},
				Entry("MAINTENANCE->MAINTENANCE", MAINTENANCE, MAINTENANCE),
				Entry("MAINTENANCE->RUN", MAINTENANCE, RUN),
				Entry("MAINTENANCE->FAIL", MAINTENANCE, FAIL),
				Entry("IDLE->MAINTENANCE", IDLE, MAINTENANCE),
				Entry("IDLE->RUN", IDLE, RUN),
				Entry("IDLE->FAIL", IDLE, FAIL),
				Entry("FAIL->MAINTENANCE", FAIL, MAINTENANCE),
				Entry("FAIL->RUN", FAIL, RUN),
				Entry("FAIL->FAIL", FAIL, FAIL),
			)
			DescribeTable("Should change state and call OnWorkerIdle",
				func(from, to WorkerState) {
					set(from)
					wc.EXPECT().OnWorkerIdle(worker)
					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
				},
				Entry("MAINTENANCE->IDLE", MAINTENANCE, IDLE),
				Entry("IDLE->IDLE", IDLE, IDLE),
				Entry("RUN->IDLE", RUN, IDLE),
				Entry("FAIL->IDLE", FAIL, IDLE),
			)
			DescribeTable("Should change state and call OnWorkerFail",
				func(from, to WorkerState) {
					set(from)
					wc.EXPECT().OnWorkerFail(worker)
					err := wl.setState(worker, to)
					Expect(err).NotTo(HaveOccurred())
					check(to)
				},
				Entry("RUN->MAINTENANCE", RUN, MAINTENANCE),
				Entry("RUN->RUN", RUN, RUN),
				Entry("RUN->FAIL", RUN, FAIL),
			)
		})
	})
	Describe("TakeBestMatchingWorker", func() {
		addWorker := func(groups Groups, caps Capabilities) *mapWorker {
			capsUUID := getUUID()
			workerUUID := WorkerUUID(capsUUID)

			caps[UUID] = capsUUID
			wl.Register(caps, dryadAddr.String(), sshdAddr.String())
			wl.mutex.RLock()
			w, ok := wl.workers[workerUUID]
			wl.mutex.RUnlock()
			Expect(ok).To(BeTrue())
			Expect(w.State).To(Equal(MAINTENANCE))

			err := wl.SetGroups(workerUUID, groups)
			Expect(err).NotTo(HaveOccurred())

			return w
		}
		addIdleWorker := func(groups Groups, caps Capabilities) *mapWorker {
			w := addWorker(groups, caps)

			err := wl.PrepareWorker(w.WorkerUUID, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(w.State).To(Equal(IDLE))

			return w
		}
		generateGroups := func(count int) Groups {
			var groups Groups
			for i := 0; i < count; i++ {
				groups = append(groups, Group(fmt.Sprintf("testGroup_%d", i)))
			}
			return groups
		}
		generateCaps := func(count int) Capabilities {
			caps := make(Capabilities)
			for i := 0; i < count; i++ {
				k := fmt.Sprintf("testCapKey_%d", i)
				v := fmt.Sprintf("testCapValue_%d", i)
				caps[k] = v
			}
			return caps
		}
		It("should fail to find matching worker when there are no workers", func() {
			ret, err := wl.TakeBestMatchingWorker(Groups{}, Capabilities{})
			Expect(err).To(Equal(ErrNoMatchingWorker))
			Expect(ret).To(BeZero())
		})
		It("should match fitting worker and set it into RUN state", func() {
			w := addIdleWorker(Groups{}, Capabilities{})

			ret, err := wl.TakeBestMatchingWorker(Groups{}, Capabilities{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ret).To(Equal(w.WorkerUUID))
			Expect(w.State).To(Equal(RUN))
		})
		It("should not match not IDLE workers", func() {
			addWorker(Groups{}, Capabilities{})

			ret, err := wl.TakeBestMatchingWorker(Groups{}, Capabilities{})
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
				Expect(w.State).To(Equal(RUN))
			}
			ret, err := wl.TakeBestMatchingWorker(generateGroups(1), generateCaps(1))
			Expect(err).To(Equal(ErrNoMatchingWorker))
			Expect(ret).To(BeZero())

			leftWorkers := []*mapWorker{w2g0c, w0g2c}
			for _, w := range leftWorkers {
				Expect(w.State).To(Equal(IDLE))
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
