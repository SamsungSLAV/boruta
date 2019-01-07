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
	"testing"

	"github.com/SamsungSLAV/boruta"
)

var validReqs = []boruta.ReqInfo{
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

func TestNewSorter(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)

	sorter, err := newSorter(nil)
	assert.Nil(err)
	assert.Equal("", sorter.item)
	assert.Equal(boruta.SortOrderAsc, sorter.order)

	si := new(boruta.SortInfo)
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("", sorter.item)
	assert.Equal(boruta.SortOrderAsc, sorter.order)

	si.Item = "foobar"
	si.Order = boruta.SortOrderAsc
	sorter, err = newSorter(si)
	assert.Equal(boruta.ErrWrongSortItem, err)
	assert.Nil(sorter)

	si.Item = ""
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("", sorter.item)
	assert.Equal(boruta.SortOrderAsc, sorter.order)

	si.Item = "id"
	si.Order = boruta.SortOrderDesc
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("id", sorter.item)
	assert.Equal(boruta.SortOrderDesc, sorter.order)

	si.Item = "priority"
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("priority", sorter.item)
	assert.Equal(boruta.SortOrderDesc, sorter.order)

	si.Item = "DeadLine"
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("deadline", sorter.item)
	assert.Equal(boruta.SortOrderDesc, sorter.order)

	si.Item = "validafter"
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("validafter", sorter.item)
	assert.Equal(boruta.SortOrderDesc, sorter.order)

	si.Item = "STATE"
	sorter, err = newSorter(si)
	assert.Nil(err)
	assert.Equal("state", sorter.item)
	assert.Equal(boruta.SortOrderDesc, sorter.order)
}

func TestSorterLen(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	s := new(sorter)
	s.reqs = make([]boruta.ReqInfo, len(validReqs))
	copy(s.reqs, validReqs)
	assert.Equal(len(validReqs), s.Len())
}

func TestSorterSwap(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	s := new(sorter)
	s.reqs = make([]boruta.ReqInfo, len(validReqs))
	copy(s.reqs, validReqs)
	s.Swap(0, 1)
	assert.Equal(s.reqs[0], validReqs[1])
	assert.Equal(s.reqs[1], validReqs[0])
}

func TestSorterLess(t *testing.T) {
	assert, rqueue, ctrl, _ := initTest(t)
	defer finiTest(rqueue, ctrl)
	s := new(sorter)
	s.reqs = make([]boruta.ReqInfo, len(validReqs))
	copy(s.reqs, validReqs)
	s.item = "panic"
	assert.Panics(func() { s.Less(0, 1) })

	s.item = ""
	assert.True(s.Less(0, 1))
	s.order = boruta.SortOrderDesc
	assert.False(s.Less(0, 1))

	s.item = "id"
	assert.False(s.Less(0, 1))
	s.order = boruta.SortOrderAsc
	assert.True(s.Less(0, 1))

	s.item = "priority"
	assert.True(s.Less(1, 2))
	// equal priorities, sort by ID
	assert.True(s.Less(0, 2))
	s.order = boruta.SortOrderDesc
	assert.False(s.Less(1, 2))
	// equal priorities, sort by ID
	assert.False(s.Less(0, 2))

	s.item = "deadline"
	assert.False(s.Less(1, 2))
	// equal deadlines, sort by ID
	assert.False(s.Less(0, 1))
	s.order = boruta.SortOrderAsc
	assert.True(s.Less(1, 2))
	// equal deadlines, sort by ID
	assert.True(s.Less(0, 1))

	s.item = "validafter"
	assert.True(s.Less(0, 2))
	// equal validafters, sort by ID
	assert.True(s.Less(0, 3))
	s.order = boruta.SortOrderDesc
	assert.False(s.Less(0, 2))
	// equal validafters, sort by ID
	assert.False(s.Less(0, 3))

	s.item = "state"
	assert.True(s.Less(0, 1))
	// equal states, sort by ID
	assert.False(s.Less(0, 3))
	s.order = boruta.SortOrderAsc
	assert.False(s.Less(0, 1))
	// equal states, sort by ID
	assert.True(s.Less(0, 3))
}
