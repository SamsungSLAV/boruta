/*
 *  Copyright (c) 2017-2022 Samsung Electronics Co., Ltd All Rights Reserved
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
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/suite"
)

const loopRoutineName = "github.com/SamsungSLAV/boruta/requests.(*requestTimes).loop"

func countGoRoutine(name string) int {
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

func getLen(t *requestTimes) int {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.times.Len()
}

func getMin(t *requestTimes) requestTime {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.times.Min()
}

func prepareRequestTime(after time.Duration, req boruta.ReqID) requestTime {
	d := time.Duration(after)
	n := time.Now().Add(d)
	return requestTime{time: n, req: req}
}

func chanIsOpen(ch chan bool) bool {
	if ch == nil {
		return false
	}
	select {
	case _, open := <-ch:
		return open
	default:
		return true
	}
}

type TestMatcher struct {
	Counter  int
	Notified []boruta.ReqID
}

func (m *TestMatcher) Notify(reqs []boruta.ReqID) {
	m.Counter++
	if m.Notified == nil {
		m.Notified = make([]boruta.ReqID, 0)
	}
	m.Notified = append(m.Notified, reqs...)
}

type TimesTestSuite struct {
	suite.Suite
	t         *requestTimes
	baseCount int
}

func (s *TimesTestSuite) SetupSuite() {
	debug.SetGCPercent(1)
}

func (s *TimesTestSuite) SetupTest() {
	s.baseCount = countGoRoutine(loopRoutineName)
	s.t = newRequestTimes()
}

func (s *TimesTestSuite) TearDownTest() {
	if s.t != nil {
		s.t.finish()
	}
	s.Equal(s.baseCount, countGoRoutine(loopRoutineName))
}

func (s *TimesTestSuite) TearDownSuite() {
	p := 100
	env, found := os.LookupEnv("GOGC")
	if found {
		tmp, err := strconv.Atoi(env)
		if err == nil {
			p = tmp
		}
	}
	debug.SetGCPercent(p)
}

func (s *TimesTestSuite) TestNewRequestTimes() {
	// newRequestTimes shoul return valid, zeroed instance
	assertProperlyInitialized := func(t *requestTimes) {
		s.T().Helper()
		s.NotNil(t)
		s.NotNil(t.times)
		s.Zero(getLen(t))
		s.NotNil(t.timer)
		s.Nil(t.matcher)
		s.NotNil(t.mutex)
		s.True(chanIsOpen(t.stop))
	}

	assertProperlyInitialized(s.t)
	s.Equal(s.baseCount+1, countGoRoutine(loopRoutineName))

	t := newRequestTimes()
	defer t.finish()
	assertProperlyInitialized(t)
	s.NotSame(s.t, t)
	s.NotSame(s.t.times, t.times)
	s.NotSame(s.t.timer, t.timer)
	s.NotSame(s.t.mutex, t.mutex)
	s.NotSame(s.t.stop, t.stop)
	s.Equal(s.baseCount+2, countGoRoutine(loopRoutineName))
}

func (s *TimesTestSuite) TestFinish() {
	finishAndCheck := func() {
		s.T().Helper()
		s.t.finish()
		s.False(chanIsOpen(s.t.stop))
		s.Equal(s.baseCount, countGoRoutine(loopRoutineName))
	}

	s.Run("Empty", func() {
		s.Zero(getLen(s.t))
		finishAndCheck()
	})

	s.Run("NotEmpty", func() {
		s.SetupTest()
		s.t.insert(prepareRequestTime(time.Minute, 1))
		s.t.insert(prepareRequestTime(time.Hour, 2))
		s.t.insert(prepareRequestTime(0, 3))
		s.NotZero(getLen(s.t))
		finishAndCheck()
		s.t = nil // Don't call finish in test teardown.
	})
}

func (s *TimesTestSuite) TestInsert() {
	r100m := prepareRequestTime(100*time.Millisecond, 100)
	// Check one element.
	s.Run("OneElement", func() {
		s.t.insert(r100m)
		s.Equal(1, getLen(s.t))
		s.Equal(r100m, getMin(s.t))
		s.Eventually(func() bool { return getLen(s.t) == 0 }, 120*time.Millisecond, 10*time.Millisecond)
	})

	s.Run("ManyElements", func() {
		r200m := prepareRequestTime(200*time.Millisecond, 200)
		r500m := prepareRequestTime(500*time.Millisecond, 500)
		r800m := prepareRequestTime(800*time.Millisecond, 800)
		s.t.insert(r200m)
		s.t.insert(r100m)
		s.t.insert(r800m)
		s.t.insert(r100m)

		length := getLen(s.t)
		s.Equal(r100m, getMin(s.t))
		s.Equal(4, length)

		// Expect process() to remove 2 elements after 100 ms [100 ms].
		length -= 2
		s.Eventually(func() bool { return getLen(s.t) == length }, 110*time.Millisecond, 10*time.Millisecond)
		s.Equal(r200m, getMin(s.t))

		// Expect process() to remove 1 element after another 100 ms [200 ms].
		length -= 1
		s.Eventually(func() bool { return getLen(s.t) == length }, 210*time.Millisecond, 10*time.Millisecond)
		s.Equal(r800m, getMin(s.t))

		// Add another element.
		s.t.insert(r500m)
		length++
		s.Eventually(func() bool { return getLen(s.t) == length }, 100*time.Millisecond, 10*time.Millisecond)
		s.Equal(r500m, getMin(s.t))

		// Expect process() to remove 1 element after another 300 ms [500 ms].
		length--
		s.Eventually(func() bool { return getLen(s.t) == length }, 310*time.Millisecond, 10*time.Millisecond)
		s.Equal(r800m, getMin(s.t))

		// Expect process() to remove 1 element after another 300 ms [800 ms].
		s.Eventually(func() bool { return getLen(s.t) == 0 }, 310*time.Millisecond, 10*time.Millisecond)
	})
}

func (s *TimesTestSuite) TestSetMatcher() {
	var m TestMatcher

	s.Nil(s.t.matcher)
	s.t.setMatcher(&m)
	s.t.mutex.Lock()
	s.Same(&m, s.t.matcher)
	s.Zero(m.Counter)
	s.t.mutex.Unlock()

	rid := boruta.ReqID(100)
	s.t.insert(prepareRequestTime(100*time.Millisecond, rid))
	s.t.mutex.Lock()
	s.Zero(m.Counter)
	s.Nil(m.Notified)
	s.t.mutex.Unlock()

	// process() should remove element after 100ms.
	s.Eventually(func() bool { return getLen(s.t) == 0 }, 120*time.Millisecond, 10*time.Millisecond)
	s.t.mutex.Lock()
	s.Equal(1, m.Counter)
	s.Equal(1, len(m.Notified))
	s.Contains(m.Notified, rid)
	s.t.mutex.Unlock()
}

func (s *TimesTestSuite) TestProcess() {
	var m TestMatcher
	reqs := [...]boruta.ReqID{101, 102, 103, 104, 105}
	r100m := prepareRequestTime(100*time.Millisecond, 0)

	s.t.setMatcher(&m)
	for _, r := range reqs {
		r100m.req = r
		s.t.insert(r100m)
	}
	s.t.mutex.Lock()
	s.Zero(m.Counter)
	s.t.mutex.Unlock()

	// Expect process() to remove all elements after 100 ms [100 ms].
	s.Eventually(func() bool { return getLen(s.t) == 0 }, 120*time.Millisecond, 10*time.Millisecond)
	s.t.mutex.Lock()
	s.Equal(1, m.Counter)
	s.Equal(len(reqs), len(m.Notified))
	s.ElementsMatch(reqs, m.Notified)
	s.t.mutex.Unlock()
}

func (s *TimesTestSuite) TestPastTime() {
	var m TestMatcher
	s.t.setMatcher(&m)

	rid := boruta.ReqID(200)
	s.t.insert(prepareRequestTime(-time.Hour, rid))

	// Expect process() to remove element.
	s.Eventually(func() bool { return getLen(s.t) == 0 }, 100*time.Millisecond, 10*time.Millisecond)
	s.t.mutex.Lock()
	s.Equal(1, m.Counter)
	s.Equal(1, len(m.Notified))
	s.Contains(m.Notified, rid)
	s.t.mutex.Unlock()
}

func TestTimesTestSuite(t *testing.T) {
	suite.Run(t, new(TimesTestSuite))
}
