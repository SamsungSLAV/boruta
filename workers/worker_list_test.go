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

package workers

import (
	. "git.tizen.org/tools/boruta"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
)

var _ = Describe("WorkerList", func() {
	var wl *WorkerList
	BeforeEach(func() {
		wl = NewWorkerList()
	})

	Describe("Register", func() {
		var registeredWorkers []string

		BeforeEach(func() {
			registeredWorkers = make([]string, 0)
		})

		compareLists := func() {
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
			err := wl.Register(nil)
			Expect(err).To(Equal(ErrMissingUUID))
		})

		getRandomCaps := func() Capabilities {
			return Capabilities{
				UUID: uuid.NewV4().String(),
			}
		}

		It("should add Worker in MAINTENANCE state", func() {
			caps := getRandomCaps()
			err := wl.Register(caps)
			Expect(err).ToNot(HaveOccurred())
			uuid := WorkerUUID(caps[UUID])
			Expect(wl.workers).To(HaveKey(uuid))
			Expect(wl.workers[uuid].State).To(Equal(MAINTENANCE))
		})

		It("should update the caps when called twice for the same worker", func() {
			var err error
			Expect(wl.workers).To(BeEmpty())
			caps := getRandomCaps()

			By("registering worker")
			err = wl.Register(caps)
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps[UUID])
			compareLists()

			By("updating the caps")
			caps["test-key"] = "test-value"
			err = wl.Register(caps)
			Expect(err).ToNot(HaveOccurred())
			Expect(wl.workers[WorkerUUID(caps[UUID])].Caps).To(Equal(caps))
			compareLists()
		})

		It("should work when called once", func() {
			var err error
			Expect(wl.workers).To(BeEmpty())
			caps := getRandomCaps()

			By("registering first worker")
			err = wl.Register(caps)
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps[UUID])
			compareLists()
		})

		It("should work when called twice with different caps", func() {
			var err error
			Expect(wl.workers).To(BeEmpty())
			caps1 := getRandomCaps()
			caps2 := getRandomCaps()

			By("registering first worker")
			err = wl.Register(caps1)
			Expect(err).ToNot(HaveOccurred())
			registeredWorkers = append(registeredWorkers, caps1[UUID])
			compareLists()

			By("registering second worker")
			err = wl.Register(caps2)
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
				newUUID = WorkerUUID(uuid.NewV4().String())
			}
			return newUUID
		}
		registerWorker := func() WorkerUUID {
			capsUUID := uuid.NewV4().String()
			err := wl.Register(Capabilities{UUID: capsUUID})
			Expect(err).ToNot(HaveOccurred())
			Expect(wl.workers).ToNot(BeEmpty())
			return WorkerUUID(capsUUID)
		}

		BeforeEach(func() {
			Expect(wl.workers).To(BeEmpty())
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
					wl.workers[worker].State = state
					err := wl.SetFail(worker, "")
					Expect(err).ToNot(HaveOccurred())
					Expect(wl.workers[worker].State).To(Equal(FAIL))
				}
			})

			It("Should fail to SetFail in MAINTENANCE state", func() {
				Expect(wl.workers[worker].State).To(Equal(MAINTENANCE))
				err := wl.SetFail(worker, "")
				Expect(err).To(Equal(ErrInMaintenance))
				Expect(wl.workers[worker].State).To(Equal(MAINTENANCE))
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
				Expect(wl.workers).ToNot(HaveKey(worker))
			})

			It("should fail to deregister same worker twice", func() {
				err := wl.Deregister(worker)
				Expect(err).ToNot(HaveOccurred())
				Expect(wl.workers).ToNot(HaveKey(worker))

				err = wl.Deregister(worker)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should fail to deregister worker not in MAINTENANCE state", func() {
				for _, state := range []WorkerState{IDLE, RUN, FAIL} {
					wl.workers[worker].State = state
					err := wl.Deregister(worker)
					Expect(err).To(Equal(ErrNotInMaintenance))
					Expect(wl.workers).To(HaveKey(worker))
				}
			})
		})

		Describe("SetState", func() {
			It("should fail to SetState of nonexistent worker", func() {
				uuid := randomUUID()
				err := wl.SetState(uuid, MAINTENANCE)
				Expect(err).To(Equal(ErrWorkerNotFound))
			})

			It("should work to SetState for valid transitions", func() {
				validTransitions := [][]WorkerState{
					{MAINTENANCE, IDLE},
					{IDLE, MAINTENANCE},
					{RUN, MAINTENANCE},
					{FAIL, MAINTENANCE},
				}
				for _, transition := range validTransitions {
					fromState, toState := transition[0], transition[1]
					wl.workers[worker].State = fromState
					err := wl.SetState(worker, toState)
					Expect(err).ToNot(HaveOccurred())
					Expect(wl.workers[worker].State).To(Equal(toState))
				}
			})

			It("should fail to SetState for invalid transitions", func() {
				invalidTransitions := [][]WorkerState{
					{RUN, IDLE},
					{FAIL, IDLE},
				}
				for _, transition := range invalidTransitions {
					fromState, toState := transition[0], transition[1]
					wl.workers[worker].State = fromState
					err := wl.SetState(worker, toState)
					Expect(err).To(Equal(ErrForbiddenStateChange))
					Expect(wl.workers[worker].State).To(Equal(fromState))
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
					wl.workers[worker].State = fromState
					err := wl.SetState(worker, toState)
					Expect(err).To(Equal(ErrWrongStateArgument))
					Expect(wl.workers[worker].State).To(Equal(fromState))
				}
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
				Expect(wl.workers[worker].Groups).To(Equal(group))

				By("setting it to nil")
				err = wl.SetGroups(worker, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(wl.workers[worker].Groups).To(BeNil())
			})
		})

		Describe("ListWorkers", func() {
			var refWorkerList []WorkerInfo

			registerAndSetGroups := func(groups Groups, caps Capabilities) WorkerInfo {
				capsUUID := uuid.NewV4().String()
				caps[UUID] = capsUUID
				err := wl.Register(caps)
				Expect(err).ToNot(HaveOccurred())
				workerID := WorkerUUID(capsUUID)

				err = wl.SetGroups(workerID, groups)
				Expect(err).ToNot(HaveOccurred())

				return wl.workers[workerID].WorkerInfo
			}

			BeforeEach(func() {
				refWorkerList = make([]WorkerInfo, 1)
				// Add worker with minimal caps and empty groups.
				refWorkerList[0] = wl.workers[worker].WorkerInfo
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
				Expect(workerInfo).To(Equal(wl.workers[worker].WorkerInfo))
			})
		})
	})
})
