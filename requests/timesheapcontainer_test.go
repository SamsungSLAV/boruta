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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TimesHeapContainerTestSuite struct {
	suite.Suite
	thc            timesHeapContainer
	t0, t1, t2, t3 time.Time
	thcLen         int
}

func (s *TimesHeapContainerTestSuite) SetupTest() {
	s.t0 = time.Now()
	s.t1 = s.t0.AddDate(0, 0, 1)
	s.t2 = s.t1.AddDate(0, 0, 1)
	s.t3 = s.t2.AddDate(0, 0, 1)

	s.thc = []requestTime{
		{time: s.t0, req: 1},
		{time: s.t3, req: 2},
		{time: s.t1, req: 3},
		{time: s.t2, req: 4},
		{time: s.t1, req: 1},
		{time: s.t3, req: 2},
	}
	s.thcLen = len(s.thc)
}

func (s *TimesHeapContainerTestSuite) TestLen() {
	var c timesHeapContainer
	s.Zero(c.Len())

	e := make([]requestTime, 0)
	c = e
	s.Zero(c.Len())

	s.Equal(s.thcLen, s.thc.Len())
}

func (s *TimesHeapContainerTestSuite) TestLess() {
	s.True(s.thc.Less(0, 1))  // t0 < t3
	s.False(s.thc.Less(1, 0)) // t3 < t0

	s.False(s.thc.Less(1, 2)) // t3 < t1
	s.True(s.thc.Less(2, 1))  // t1 < t3

	s.False(s.thc.Less(3, 3)) // t2 < t2

	s.False(s.thc.Less(2, 4)) // t1/3 < t1/1
	s.True(s.thc.Less(4, 2))  // t1/1 < t1/3

	s.False(s.thc.Less(1, 5)) // t3/2 < t3/2
	s.False(s.thc.Less(5, 1)) // t3/2 < t3/2
}

func (s *TimesHeapContainerTestSuite) TestSwap() {
	// swap different elements
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[1])
	s.Equal(requestTime{time: s.t2, req: 4}, s.thc[3])
	s.thc.Swap(1, 3)
	s.Equal(requestTime{time: s.t2, req: 4}, s.thc[1])
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[3])
	// restore original order
	s.thc.Swap(1, 3)
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[1])
	s.Equal(requestTime{time: s.t2, req: 4}, s.thc[3])

	// swap same values
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[1])
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[5])
	s.thc.Swap(1, 5)
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[1])
	s.Equal(requestTime{time: s.t3, req: 2}, s.thc[5])

	// swap in place
	s.Equal(requestTime{time: s.t2, req: 4}, s.thc[3])
	s.thc.Swap(3, 3)
	s.Equal(requestTime{time: s.t2, req: 4}, s.thc[3])
}

func (s *TimesHeapContainerTestSuite) TestPush() {
	// Push to empty container.
	var c timesHeapContainer
	s.Zero(c.Len())
	c.Push(requestTime{time: s.t2, req: 4})
	s.Equal(1, c.Len())
	s.Equal(requestTime{time: s.t2, req: 4}, c[0])

	// Push to non empty container.
	s.Equal(s.thcLen, s.thc.Len())
	s.thc.Push(requestTime{time: s.t0, req: 7})
	s.Equal(s.thcLen+1, s.thc.Len())
	s.Equal(requestTime{time: s.t0, req: 7}, s.thc[s.thcLen])

	s.thc.Push(requestTime{time: s.t2, req: 5})
	s.Equal(s.thcLen+2, s.thc.Len())
	s.Equal(requestTime{time: s.t2, req: 5}, s.thc[s.thcLen+1])
}

func (s *TimesHeapContainerTestSuite) TestPop() {
	// Panic when container is empty.
	s.Panics(func() {
		var c timesHeapContainer
		c.Pop()
	})

	// Pop elements from container
	s.Equal(s.thcLen, s.thc.Len())

	t := s.thc.Pop()
	s.Equal(s.thcLen-1, s.thc.Len())
	s.Equal(requestTime{time: s.t3, req: 2}, t)

	t = s.thc.Pop()
	s.Equal(s.thcLen-2, s.thc.Len())
	s.Equal(requestTime{time: s.t1, req: 1}, t)

	t = s.thc.Pop()
	s.Equal(s.thcLen-3, s.thc.Len())
	s.Equal(requestTime{time: s.t2, req: 4}, t)
}

func TestTimesHeapContainerTestSuite(t *testing.T) {
	suite.Run(t, new(TimesHeapContainerTestSuite))
}
