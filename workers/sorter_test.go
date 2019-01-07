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

package workers

import (
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

func TestNewSorter(t *testing.T) {
	tcases := [...]struct {
		name string
		info *boruta.SortInfo
		res  *sorter
		err  error
	}{
		{
			name: "uninitialized SortInfo",
			info: new(boruta.SortInfo),
			res:  &sorter{item: "", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "uuid smallcaps, ascending",
			info: &boruta.SortInfo{Item: "uuid", Order: boruta.SortOrderAsc},
			res:  &sorter{item: "uuid", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "uuid ALLCAPS, ascending",
			info: &boruta.SortInfo{Item: "UUID", Order: boruta.SortOrderAsc},
			res:  &sorter{item: "uuid", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "uuid CammelCase, ascending",
			info: &boruta.SortInfo{Item: "UuId", Order: boruta.SortOrderAsc},
			res:  &sorter{item: "uuid", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "state, ascending",
			info: &boruta.SortInfo{Item: "state", Order: boruta.SortOrderAsc},
			res:  &sorter{item: "state", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "groups, descending",
			info: &boruta.SortInfo{Item: "Groups", Order: boruta.SortOrderDesc},
			res:  &sorter{item: "groups", order: boruta.SortOrderDesc},
			err:  nil,
		},
		{
			name: "uuid, ascending",
			info: nil,
			res:  &sorter{item: "uuid", order: boruta.SortOrderAsc},
			err:  nil,
		},
		{
			name: "invalid SortInfo.Item",
			info: &boruta.SortInfo{Item: "foo", Order: boruta.SortOrderDesc},
			res:  nil,
			err:  boruta.ErrWrongSortItem,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			res, err := newSorter(tc.info)
			assert.Equal(tc.res, res, tc.name)
			assert.Equal(tc.err, err, tc.name)
		})
	}
}

func TestSorterLen(t *testing.T) {
	assert := assert.New(t)

	sorter, err := newSorter(nil)
	assert.Nil(err)
	assert.Zero(sorter.Len())

	sorter.list = []boruta.WorkerInfo{}
	assert.Equal(len(sorter.list), sorter.Len())

	for i := 0; i < 5; i++ {
		sorter.list = append(sorter.list, boruta.WorkerInfo{})
	}
	assert.Equal(len(sorter.list), sorter.Len())
}

func TestSorterSwap(t *testing.T) {
	assert := assert.New(t)

	sorter := &sorter{
		list: []boruta.WorkerInfo{
			boruta.WorkerInfo{WorkerUUID: "foo"},
			boruta.WorkerInfo{WorkerUUID: "bar"},
		},
	}
	assert.EqualValues("foo", sorter.list[0].WorkerUUID)
	assert.EqualValues("bar", sorter.list[1].WorkerUUID)
	sorter.Swap(0, 1)
	assert.EqualValues("bar", sorter.list[0].WorkerUUID)
	assert.EqualValues("foo", sorter.list[1].WorkerUUID)
}

func TestSorterLess(t *testing.T) {
	tcases := [...]struct {
		name    string
		sorter  *sorter
		workers []boruta.WorkerInfo
	}{
		{
			name: "with default sorter",
			sorter: &sorter{
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "bar",
					},
					boruta.WorkerInfo{
						WorkerUUID: "baz",
					},
				},
			},
		},
		{
			name: "by uuid",
			sorter: &sorter{
				item: "uuid",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "bar",
					},
					boruta.WorkerInfo{
						WorkerUUID: "baz",
					},
				},
			},
		},
		{
			name: "by state",
			sorter: &sorter{
				item: "state",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						State:      boruta.FAIL,
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						State:      boruta.IDLE,
					},
				},
			},
		},
		{
			name: "with equal states",
			sorter: &sorter{
				item: "state",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						State:      boruta.IDLE,
					},
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						State:      boruta.IDLE,
					},
				},
			},
		},
		{
			name: "by groups (1 set empty)",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"aaa"},
					},
				},
			},
		},
		{
			name: "by groups",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"aaa"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"bbb"},
					},
				},
			},
		},
		{
			name: "with equal groups",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"aaa"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"aaa"},
					},
				},
			},
		},
		{
			name: "by groups (different sizes of groups)",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"aaa", "bbb"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"bbb"},
					},
				},
			},
		},
		{
			name: "by groups (only one letter different)",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"aaa", "bbb"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"aab", "bbb"},
					},
				},
			},
		},
		{
			name: "by groups (prefix subset of group)",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"aaa", "bbb"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"aaa", "bbb", "ccc"},
					},
				},
			},
		},
		{
			name: "by groups (same strings with different case)",
			sorter: &sorter{
				item: "groups",
				list: []boruta.WorkerInfo{
					boruta.WorkerInfo{
						WorkerUUID: "baz",
						Groups:     boruta.Groups{"AAA", "BBB", "CCC"},
					},
					boruta.WorkerInfo{
						WorkerUUID: "bar",
						Groups:     boruta.Groups{"aaa", "bbb", "ccc"},
					},
				},
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)

			tc.sorter.order = boruta.SortOrderAsc
			assert.True(tc.sorter.Less(0, 1), tc.name)
			assert.False(tc.sorter.Less(1, 0), tc.name)

			tc.sorter.order = boruta.SortOrderDesc
			assert.False(tc.sorter.Less(0, 1), tc.name)
			assert.True(tc.sorter.Less(1, 0), tc.name)
		})
	}

	panicTCaseName := "sorter with invalid item"
	t.Run(panicTCaseName, func(t *testing.T) {
		res := sorter{item: "boom"}
		assert.Panics(t, func() { res.Less(0, 1) }, panicTCaseName)
	})
}
