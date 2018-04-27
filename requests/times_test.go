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
	"runtime"
	"runtime/debug"
	"time"

	. "git.tizen.org/tools/boruta"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type TestMatcher struct {
	Counter  int
	Notified []ReqID
}

func (m *TestMatcher) Notify(reqs []ReqID) {
	m.Counter++
	if m.Notified == nil {
		m.Notified = make([]ReqID, 0)
	}
	m.Notified = append(m.Notified, reqs...)
}

var _ = Describe("Times", func() {
	loopRoutineName := "git.tizen.org/tools/boruta/requests.(*requestTimes).loop"
	debug.SetGCPercent(1)
	var t *requestTimes
	var baseCount int

	countGoRoutine := func(name string) int {
		runtime.GC()

		counter := 0
		p := make([]runtime.StackRecord, 1)
		n, _ := runtime.GoroutineProfile(p)
		p = make([]runtime.StackRecord, 2*n)
		runtime.GoroutineProfile(p)
		for _, s := range p {
			for _, f := range s.Stack() {
				if f != 0 {
					if runtime.FuncForPC(f).Name() == name {
						counter++
					}
				}
			}
		}
		return counter
	}
	getLen := func() int {
		t.mutex.Lock()
		defer t.mutex.Unlock()
		return t.times.Len()
	}
	getMin := func() requestTime {
		t.mutex.Lock()
		defer t.mutex.Unlock()
		return t.times.Min()
	}
	prepareRequestTime := func(after time.Duration, req ReqID) requestTime {
		d := time.Duration(after)
		n := time.Now().Add(d)
		return requestTime{time: n, req: req}
	}

	BeforeEach(func() {
		baseCount = countGoRoutine(loopRoutineName)
		t = newRequestTimes()
	})
	AfterEach(func() {
		if t != nil {
			t.finish()
		}
		Expect(countGoRoutine(loopRoutineName)).To(Equal(baseCount))
	})
	Describe("newRequestTimes", func() {
		It("should init all fields", func() {
			Expect(t).NotTo(BeNil(), "t")
			Expect(t.times).NotTo(BeNil())
			Expect(t.times.Len()).To(BeZero())
			Expect(t.timer).NotTo(BeNil())
			Expect(t.matcher).To(BeNil())
			Expect(t.mutex).NotTo(BeNil())
			Expect(t.stop).NotTo(BeClosed())
			Expect(countGoRoutine(loopRoutineName)).To(Equal(baseCount + 1))
		})
		It("should create separate object in every call", func() {
			t2 := newRequestTimes()
			defer t2.finish()

			Expect(t).NotTo(BeIdenticalTo(t2))
			Expect(t.times).NotTo(BeIdenticalTo(t2.times))
			Expect(t.timer).NotTo(BeIdenticalTo(t2.timer))
			Expect(t.mutex).NotTo(BeIdenticalTo(t2.mutex))
			Expect(t.stop).NotTo(BeIdenticalTo(t2.stop))
			Expect(countGoRoutine(loopRoutineName)).To(Equal(baseCount + 2))
		})
	})
	Describe("finish", func() {
		It("should work with unused empty structure", func() {
			t.finish()

			Expect(t.stop).To(BeClosed())
			Expect(countGoRoutine(loopRoutineName)).To(Equal(baseCount))
			// Avoid extra finish in AfterEach.
			t = nil
		})
		It("should work when times heap is not empty", func() {
			t.insert(prepareRequestTime(time.Minute, 1))
			t.insert(prepareRequestTime(time.Hour, 2))
			t.insert(prepareRequestTime(0, 3))

			t.finish()

			Expect(t.stop).To(BeClosed())
			Expect(countGoRoutine(loopRoutineName)).To(Equal(baseCount))
			// Avoid extra finish in AfterEach.
			t = nil
		})
	})
	Describe("insert", func() {
		It("should insert single time", func() {
			r100m := prepareRequestTime(100*time.Millisecond, 100)

			t.insert(r100m)
			Expect(getLen()).To(Equal(1))
			Expect(getMin()).To(Equal(r100m))

			Eventually(getLen).Should(BeZero())
		})
		It("should insert multiple times", func() {
			r100m := prepareRequestTime(100*time.Millisecond, 100)
			r200m := prepareRequestTime(200*time.Millisecond, 200)
			r500m := prepareRequestTime(500*time.Millisecond, 500)
			r800m := prepareRequestTime(800*time.Millisecond, 800)

			t.insert(r100m)
			t.insert(r200m)
			t.insert(r100m)
			t.insert(r800m)
			Expect(getLen()).To(Equal(4))
			Expect(getMin()).To(Equal(r100m))

			// Expect process() to remove 2 elements after 100 ms [100 ms].
			Eventually(getLen).Should(Equal(2))
			Expect(getMin()).To(Equal(r200m))

			// Expect process() to remove 1 element after another 100 ms [200 ms].
			Eventually(getLen).Should(Equal(1))
			Expect(getMin()).To(Equal(r800m))

			t.insert(r500m)
			Expect(getLen()).To(Equal(2))
			Expect(getMin()).To(Equal(r500m))

			// Expect process() to remove 1 element after another 300 ms [500 ms].
			Eventually(getLen).Should(Equal(1))
			Expect(getMin()).To(Equal(r800m))

			// Expect process() to remove 1 element after another 300 ms [800 ms].
			Eventually(getLen).Should(BeZero())
		})
	})
	Describe("setMatcher", func() {
		It("should set matcher", func() {
			var m TestMatcher

			Expect(t.matcher).To(BeNil())
			t.setMatcher(&m)
			Expect(t.matcher).To(Equal(&m))
			Expect(m.Counter).To(BeZero())
		})
		It("should notify matcher", func() {
			var m TestMatcher
			t.setMatcher(&m)

			rid := ReqID(100)
			t.insert(prepareRequestTime(100*time.Millisecond, rid))

			Expect(m.Counter).To(BeZero())
			Expect(m.Notified).To(BeNil())

			// Expect process() to remove 1 element after 100 ms [100 ms].
			Eventually(getLen).Should(BeZero())
			Expect(m.Counter).To(Equal(1))
			Expect(len(m.Notified)).To(Equal(1))
			Expect(m.Notified).To(ContainElement(rid))

		})
	})
	Describe("process", func() {
		It("should be run once for same times", func() {
			var m TestMatcher
			r100m := prepareRequestTime(100*time.Millisecond, 0)
			reqs := []ReqID{101, 102, 103, 104, 105}

			t.setMatcher(&m)
			for _, r := range reqs {
				r100m.req = r
				t.insert(r100m)
			}
			Expect(m.Counter).To(BeZero())

			// Expect process() to remove all elements after 100 ms [100 ms].
			Eventually(getLen).Should(BeZero())
			Expect(m.Counter).To(Equal(1))
			Expect(len(m.Notified)).To(Equal(len(reqs)))
			Expect(m.Notified).To(ConsistOf(reqs))
		})
	})
	Describe("past time", func() {
		It("should handle times in the past properly", func() {
			var m TestMatcher
			t.setMatcher(&m)

			rid := ReqID(200)
			t.insert(prepareRequestTime(-time.Hour, rid))

			// Expect process() to remove element.
			Eventually(getLen).Should(BeZero())
			Expect(m.Counter).To(Equal(1))
			Expect(len(m.Notified)).To(Equal(1))
			Expect(m.Notified).To(ContainElement(rid))
		})
	})
})
