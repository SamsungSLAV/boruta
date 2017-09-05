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

var _ = Describe("TimesHeapContainer", func() {
	var thc timesHeapContainer
	var t0, t1, t2, t3 time.Time
	var thcLen int
	BeforeEach(func() {
		t0 = time.Now()
		t1 = t0.AddDate(0, 0, 1)
		t2 = t1.AddDate(0, 0, 1)
		t3 = t2.AddDate(0, 0, 1)

		thc = []requestTime{
			{time: t0, req: 1},
			{time: t3, req: 2},
			{time: t1, req: 3},
			{time: t2, req: 4},
			{time: t1, req: 1},
			{time: t3, req: 2},
		}
		thcLen = len(thc)
	})

	Describe("Len", func() {
		It("should work with nil container", func() {
			var c timesHeapContainer
			Expect(c.Len()).To(BeZero())
		})
		It("should work with empty container", func() {
			s := make([]requestTime, 0)
			var c timesHeapContainer = s
			Expect(c.Len()).To(BeZero())
		})
		It("should work with non-empty container", func() {
			Expect(thc.Len()).To(Equal(thcLen))
		})
	})

	Describe("Less", func() {
		It("should compare elements", func() {
			Expect(thc.Less(0, 1)).To(BeTrue())  //t0 < t3
			Expect(thc.Less(1, 0)).To(BeFalse()) //t3 < t0

			Expect(thc.Less(1, 2)).To(BeFalse()) //t3 < t1
			Expect(thc.Less(2, 1)).To(BeTrue())  //t1 < t3

			Expect(thc.Less(3, 3)).To(BeFalse()) //t2 < t2

			Expect(thc.Less(2, 4)).To(BeFalse()) //t1/3 < t1/1
			Expect(thc.Less(4, 2)).To(BeTrue())  //t1/1 < t1/3

			Expect(thc.Less(1, 5)).To(BeFalse()) //t3/2 < t3/2
			Expect(thc.Less(5, 1)).To(BeFalse()) //t3/2 < t3/2
		})
	})

	Describe("Swap", func() {
		It("should swap different elements", func() {
			Expect(thc[1]).To(Equal(requestTime{time: t3, req: 2}))
			Expect(thc[3]).To(Equal(requestTime{time: t2, req: 4}))
			thc.Swap(1, 3)
			Expect(thc[1]).To(Equal(requestTime{time: t2, req: 4}))
			Expect(thc[3]).To(Equal(requestTime{time: t3, req: 2}))
		})
		It("should swap same values", func() {
			Expect(thc[1]).To(Equal(requestTime{time: t3, req: 2}))
			Expect(thc[5]).To(Equal(requestTime{time: t3, req: 2}))
			thc.Swap(1, 5)
			Expect(thc[1]).To(Equal(requestTime{time: t3, req: 2}))
			Expect(thc[5]).To(Equal(requestTime{time: t3, req: 2}))
		})
		It("should swap in place", func() {
			Expect(thc[3]).To(Equal(requestTime{time: t2, req: 4}))
			thc.Swap(3, 3)
			Expect(thc[3]).To(Equal(requestTime{time: t2, req: 4}))
		})
	})

	Describe("Push", func() {
		It("should work with empty container", func() {
			var c timesHeapContainer
			Expect(c.Len()).To(BeZero())

			c.Push(requestTime{time: t2, req: 4})

			Expect(c.Len()).To(Equal(1))
			Expect(c[0]).To(Equal(requestTime{time: t2, req: 4}))
		})
		It("should add elements to non empty container", func() {
			Expect(thc.Len()).To(Equal(thcLen))

			thc.Push(requestTime{time: t0, req: 7})

			Expect(thc.Len()).To(Equal(thcLen + 1))
			Expect(thc[thcLen]).To(Equal(requestTime{time: t0, req: 7}))

			thc.Push(requestTime{time: t2, req: 5})

			Expect(thc.Len()).To(Equal(thcLen + 2))
			Expect(thc[thcLen+1]).To(Equal(requestTime{time: t2, req: 5}))
		})
	})

	Describe("Pop", func() {
		It("should pop elements from container", func() {
			Expect(thc.Len()).To(Equal(thcLen))

			t := thc.Pop()

			Expect(thc.Len()).To(Equal(thcLen - 1))
			Expect(t).To(Equal(requestTime{time: t3, req: 2}))

			t = thc.Pop()

			Expect(thc.Len()).To(Equal(thcLen - 2))
			Expect(t).To(Equal(requestTime{time: t1, req: 1}))

			t = thc.Pop()

			Expect(thc.Len()).To(Equal(thcLen - 3))
			Expect(t).To(Equal(requestTime{time: t2, req: 4}))
		})
		It("should panic if container is empty", func() {
			Expect(func() {
				var c timesHeapContainer
				c.Pop()
			}).Should(Panic())
		})
	})
})
