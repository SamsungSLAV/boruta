/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
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
	"strings"
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SorterTestSuite struct {
	suite.Suite
	sorter *sorter
}

var validReqs = [...]boruta.ReqInfo{
	{
		ID:         1,
		Priority:   4,
		ValidAfter: now,
		Deadline:   tomorrow,
		State:      boruta.WAIT,
	},
	{
		ID:         2,
		Priority:   2,
		ValidAfter: yesterday,
		Deadline:   tomorrow,
		State:      boruta.CANCEL,
	},
	{
		ID:         3,
		Priority:   4,
		ValidAfter: tomorrow,
		Deadline:   nextWeek,
		State:      boruta.INPROGRESS,
	},
	{
		ID:         4,
		Priority:   8,
		ValidAfter: now,
		Deadline:   tomorrow,
		State:      boruta.WAIT,
	},
}

func (s *SorterTestSuite) SetupTest() {
	s.sorter = new(sorter)
	s.sorter.reqs = make([]boruta.ReqInfo, len(validReqs))
	copy(s.sorter.reqs, validReqs[:])
}

func TestNewSorter(t *testing.T) {
	assert := assert.New(t)

	testCases := [...]struct {
		name  string
		si    *boruta.SortInfo
		order boruta.SortOrder
		err   error
	}{
		{name: "NilSortInfo", si: nil, order: boruta.SortOrderAsc, err: nil},
		{name: "EmptySortInfo", si: new(boruta.SortInfo), order: boruta.SortOrderAsc, err: nil},
		{name: "EmptySortItem", si: &boruta.SortInfo{Item: "", Order: boruta.SortOrderAsc}, order: boruta.SortOrderAsc, err: nil},
		{name: "IdSortItem", si: &boruta.SortInfo{Item: "id", Order: boruta.SortOrderDesc}, order: boruta.SortOrderDesc, err: nil},
		{
			name:  "PrioritySortItem",
			si:    &boruta.SortInfo{Item: "priority", Order: boruta.SortOrderDesc},
			order: boruta.SortOrderDesc,
			err:   nil,
		},
		{
			name:  "CaseInsencitiveSortItem",
			si:    &boruta.SortInfo{Item: "DeadLine", Order: boruta.SortOrderDesc},
			order: boruta.SortOrderDesc,
			err:   nil,
		},
		{
			name:  "ValidAfterSortItem",
			si:    &boruta.SortInfo{Item: "ValidAfter", Order: boruta.SortOrderAsc},
			order: boruta.SortOrderAsc,
			err:   nil,
		},
		{
			name:  "StateSortItem",
			si:    &boruta.SortInfo{Item: "STATE", Order: boruta.SortOrderAsc},
			order: boruta.SortOrderAsc,
			err:   nil,
		},
		{
			name:  "WrongSortItem",
			si:    &boruta.SortInfo{Item: "foobar", Order: boruta.SortOrderAsc},
			order: boruta.SortOrderAsc,
			err:   boruta.ErrWrongSortItem,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			sorter, err := newSorter(test.si)
			assert.Equalf(test.err, err, "test case: %s", test.name)
			if err != nil {
				assert.Nilf(sorter, "test case: %s", test.name)
			} else {
				assert.Equalf(test.order, sorter.order, "test case: %s", test.name)
				if test.si != nil {
					assert.Equalf(strings.ToLower(test.si.Item), sorter.item, "test case: %s", test.name)
				}
			}
		})
	}
}

func (s *SorterTestSuite) TestSorterLen() {
	s.Equal(len(validReqs), s.sorter.Len())
}

func (s *SorterTestSuite) TestSorterSwap() {
	s.sorter.Swap(0, 1)
	s.Equal(s.sorter.reqs[0], validReqs[1])
	s.Equal(s.sorter.reqs[1], validReqs[0])
}

func (s *SorterTestSuite) TestSorterLess() {
	testCases := [...]struct {
		name, item string
		i, j       int
	}{
		{name: "EmptyItem", item: "", i: 0, j: 1},
		{name: "ID", item: "id", i: 0, j: 1},
		{name: "Priority", item: "priority", i: 1, j: 2},
		{name: "PriorityEqual", item: "priority", i: 0, j: 2}, // sort by IO
		{name: "Deadline", item: "deadline", i: 1, j: 2},
		{name: "DeadlineEqual", item: "deadline", i: 0, j: 1}, // sort by IO
		{name: "ValidAfter", item: "validafter", i: 0, j: 2},
		{name: "ValidAfterEqual", item: "validafter", i: 0, j: 3}, // sort by IO
		{name: "State", item: "state", i: 1, j: 0},
		{name: "StateEqual", item: "state", i: 0, j: 3}, // sort by IO
	}

	s.Run("panic", func() {
		s.sorter.item = "panic"
		s.Panics(func() { s.sorter.Less(0, 1) })
	})

	for _, test := range testCases {
		test := test
		sorter := s.sorter
		sorter.item = test.item
		s.Run(test.name, func() {
			sorter.order = boruta.SortOrderAsc
			s.Truef(sorter.Less(test.i, test.j), "test case: %s", test.name)
			sorter.order = boruta.SortOrderDesc
			s.Falsef(sorter.Less(test.i, test.j), "test case: %s", test.name)
		})
	}
}

func TestSorterTestSuite(t *testing.T) {
	suite.Run(t, new(SorterTestSuite))
}
