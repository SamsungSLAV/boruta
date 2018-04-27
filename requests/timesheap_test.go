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

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TimesHeap", func() {
	var h *timesHeap
	var t []time.Time
	var r []requestTime
	BeforeEach(func() {
		now := time.Now()
		t = make([]time.Time, 0)
		for i := 0; i < 4; i++ {
			t = append(t, now.AddDate(0, 0, i))

		}

		r = []requestTime{
			{time: t[0], req: 1},
			{time: t[1], req: 2},
			{time: t[2], req: 3},
			{time: t[3], req: 4},
			{time: t[3], req: 5},
			{time: t[1], req: 6},
		}

		h = newTimesHeap()
	})

	Describe("newTimesHeap", func() {
		It("should create an empty heap", func() {
			Expect(h).NotTo(BeNil())
			Expect(h.Len()).To(BeZero())
		})
		It("should create a new heap every time called", func() {
			h2 := newTimesHeap()

			h.Push(r[1])
			h2.Push(r[2])

			Expect(h.Len()).To(Equal(1))
			Expect(h2.Len()).To(Equal(1))
			Expect(h.Min()).To(Equal(r[1]))
			Expect(h2.Min()).To(Equal(r[2]))
		})
	})

	Describe("Len", func() {
		It("should return valid heap size", func() {
			for i, e := range r {
				h.Push(e)
				Expect(h.Len()).To(Equal(i+1), "i=%v", i)
			}

			for i := len(r) - 1; i >= 0; i-- {
				h.Pop()
				Expect(h.Len()).To(Equal(i), "i=%v", i)
			}
		})
	})

	Describe("Min", func() {
		It("should return minimum value of the heap", func() {
			toPush := []int{3, 5, 1, 2}
			pushMin := []int{3, 5, 1, 1}
			for i, e := range toPush {
				h.Push(r[e])
				Expect(h.Min()).To(Equal(r[pushMin[i]]))
			}

			popMin := []int{5, 2, 3}
			for _, e := range popMin {
				h.Pop()
				Expect(h.Min()).To(Equal(r[e]))
			}
		})
		It("should panic in case of empty heap", func() {
			Expect(func() {
				h.Min()
			}).Should(Panic())
		})
	})

	Describe("Pop", func() {
		It("should pop minimal element", func() {
			toPush := []int{5, 3, 1, 0}
			for _, e := range toPush {
				h.Push(r[e])
			}

			pushMin := []int{0, 1, 5, 3}
			for _, e := range pushMin {
				Expect(h.Min()).To(Equal(r[e]))
				Expect(h.Pop()).To(Equal(r[e]))
			}
		})
		It("should decrease heap size by one", func() {
			toPush := []int{0, 2, 1, 5}
			for _, e := range toPush {
				h.Push(r[e])
			}

			popMin := []int{0, 1, 5, 2}
			for i, e := range popMin {
				Expect(h.Len()).To(Equal(len(popMin) - i))
				Expect(h.Pop()).To(Equal(r[e]))
			}
			Expect(h.Len()).To(BeZero())
		})
		It("should panic in case of empty heap", func() {
			Expect(func() {
				h.Pop()
			}).Should(Panic())
		})
	})

	Describe("Push", func() {
		It("should add elements to the heap keeping minimum property", func() {
			toPush := []int{4, 5, 1, 3}
			pushMin := []int{4, 5, 1, 1}
			for i, e := range toPush {
				h.Push(r[e])
				Expect(h.Min()).To(Equal(r[pushMin[i]]))
			}
		})
		It("should increase heap size by one", func() {
			Expect(h.Len()).To(BeZero())
			toPush := []int{1, 0, 2, 4}
			for i, e := range toPush {
				h.Push(r[e])
				Expect(h.Len()).To(Equal(i + 1))
			}
		})
	})
})
