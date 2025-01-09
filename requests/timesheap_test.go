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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TimesHeapTestSuite struct {
	suite.Suite
	h *timesHeap
	t []time.Time
	r []requestTime
}

func (s *TimesHeapTestSuite) SetupTest() {
	now := time.Now()
	s.t = make([]time.Time, 0, 4)
	for i := 0; i < 4; i++ {
		s.t = append(s.t, now.AddDate(0, 0, i))
	}

	s.r = []requestTime{
		{time: s.t[0], req: 1},
		{time: s.t[1], req: 2},
		{time: s.t[2], req: 3},
		{time: s.t[3], req: 4},
		{time: s.t[3], req: 5},
		{time: s.t[1], req: 6},
	}

	s.h = newTimesHeap()
}

func (s *TimesHeapTestSuite) TestNewTimesHeap() {
	// Created heap should be empty.
	s.NotNil(s.h)
	s.Zero(s.h.Len())
	// It should create a new heap every time called.
	h := newTimesHeap()
	s.Equal(s.h, h)
	s.NotSame(s.h, h)
}

func (s *TimesHeapTestSuite) TestLen() {
	for i, e := range s.r {
		s.h.Push(e)
		s.Equal(i+1, s.h.Len())
	}

	for i := len(s.r) - 1; i >= 0; i-- {
		s.h.Pop()
		s.Equal(i, s.h.Len())
	}
}

func (s *TimesHeapTestSuite) TestMin() {
	// Empty heap causes panic.
	s.Panics(func() { s.h.Min() })

	// Return minimum value of the heap.
	toPush := [...]int{3, 5, 1, 2}
	pushMin := [...]int{3, 5, 1, 1}
	s.Zero(s.h.Len())
	for i, e := range toPush {
		s.h.Push(s.r[e])
		s.Equal(s.r[pushMin[i]], s.h.Min())
	}

	// Min doesn't remove elements from heap, contraty to Pop.
	s.NotZero(s.h.Len())

	popMin := [...]int{5, 2, 3}
	for _, e := range popMin {
		s.h.Pop()
		s.Equal(s.r[e], s.h.Min())
	}
}

func (s *TimesHeapTestSuite) TestPop() {
	// Empty heap causes panic.
	s.Panics(func() { s.h.Pop() })

	// Pops minial element and decreases heap size by one.
	toPush := [...]int{5, 3, 1, 0}
	for _, e := range toPush {
		s.h.Push(s.r[e])
	}
	pushMin := [...]int{0, 1, 5, 3}
	for i, e := range pushMin {
		s.Equal(s.r[e], s.h.Min())
		// Min shouldn't remove element.
		s.Equal(len(pushMin)-i, s.h.Len())
		s.Equal(s.r[e], s.h.Pop())
		// Pop should remove element.
		s.Equal(len(pushMin)-i-1, s.h.Len())
	}
	s.Zero(s.h.Len())
}

func (s *TimesHeapTestSuite) TestPush() {
	// Push adds element to heap. Size of heap is increased and minimum property is kept.
	toPush := [...]int{4, 5, 1, 3}
	pushMin := [...]int{4, 5, 1, 1}
	for i, e := range toPush {
		s.h.Push(s.r[e])
		s.Equal(s.r[pushMin[i]], s.h.Min())
		s.Equal(i+1, s.h.Len())
	}
}

func TestTimesHeapTestSuite(t *testing.T) {
	suite.Run(t, new(TimesHeapTestSuite))
}
