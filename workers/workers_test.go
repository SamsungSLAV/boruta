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

package workers_test

import (
	"math/rand"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
)

var _ = Describe("WorkerList", func() {
	var wl *workers.WorkerList
	BeforeEach(func() {
		wl = workers.NewWorkerList()
	})

	getUUID := func() string {
		u, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		return u.String()
	}

	getRandomCaps := func() boruta.Capabilities {
		return map[string]string{
			workers.UUID: getUUID(),
		}
	}

	Measure("it should be fast", func(b Benchmarker) {
		maximumWorkers := 1024
		maximumCaps := maximumWorkers / 16
		maximumGroups := maximumWorkers / 4
		matchingCaps := "TestCaps"
		caps := make([]boruta.Capabilities, maximumWorkers)
		for i := 0; i < maximumWorkers; i++ {
			caps[i] = getRandomCaps()
			caps[i][matchingCaps] = string(i % maximumCaps)
		}
		b.Time("register", func() {
			for i := 0; i < maximumWorkers; i++ {
				err := wl.Register(caps[i])
				Expect(err).ToNot(HaveOccurred())
			}
		})
		for i := 0; i < maximumWorkers; i++ {
			err := wl.SetGroups(boruta.WorkerUUID(caps[i][workers.UUID]),
				boruta.Groups{
					"TestGroup",
					boruta.Group(i % (maximumGroups / 2)),
					boruta.Group(i % maximumGroups),
				})
			Expect(err).ToNot(HaveOccurred())
		}
		maximumListTests := maximumGroups + maximumCaps
		groupWithCaps := make([]boruta.Groups, maximumListTests)
		for v := 0; v < maximumListTests; v++ {
			groupWithCaps[v] = boruta.Groups{
				boruta.Group(rand.Intn(maximumGroups)),
				boruta.Group(rand.Intn(maximumCaps)),
			}
		}
		b.Time("list all", func() {
			for i := 0; i < maximumListTests; i++ {
				_, err := wl.ListWorkers(nil, nil)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		b.Time("list with caps matching", func() {
			for i := 0; i < maximumListTests; i++ {
				_, err := wl.ListWorkers(nil,
					boruta.Capabilities{matchingCaps: string(groupWithCaps[i][1])})
				Expect(err).ToNot(HaveOccurred())
			}
		})
		b.Time("list with groups matching", func() {
			for i := 0; i < maximumListTests; i++ {
				_, err := wl.ListWorkers(boruta.Groups{boruta.Group(groupWithCaps[i][0])}, nil)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		b.Time("list with both groups and caps matching", func() {
			for i := 0; i < maximumListTests; i++ {
				_, err := wl.ListWorkers(boruta.Groups{boruta.Group(groupWithCaps[i][0])},
					boruta.Capabilities{matchingCaps: string(groupWithCaps[i][1])})
				Expect(err).ToNot(HaveOccurred())
			}
		})
		b.Time("deregister", func() {
			for i := 0; i < maximumWorkers; i++ {
				err := wl.Deregister(boruta.WorkerUUID(caps[i][workers.UUID]))
				Expect(err).ToNot(HaveOccurred())
			}
		})
	}, 2)
})
