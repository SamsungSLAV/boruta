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

// File workers/sorter.go provides implementation of boruta.Sorter interface.

package workers

import "github.com/SamsungSLAV/boruta"

// sorter implements sort.Interface. It allows to sort by worker UUID (default), State or
// Groups in ascending or descending order. It is used by ListWorkers.
type sorter struct {
	list  []boruta.WorkerInfo
	item  string
	order boruta.SortOrder
}

// newSorter returns sorter initialized with SortInfo data. It may return an error when info is nil
// or it contains unknown item.
func newSorter(info *boruta.SortInfo) (*sorter, error) {
	if info == nil {
		return &sorter{
			order: boruta.SortOrderAsc,
			item:  "uuid",
		}, nil
	}
	switch info.Item {
	case "":
	case "uuid":
	case "state":
	case "groups":
	default:
		return nil, boruta.ErrWrongSortItem
	}
	return &sorter{
		order: info.Order,
		item:  info.Item,
	}, nil
}

// Len returns length of underlying WorkerInfo slice. It is part of sort.Interface.
func (sorter *sorter) Len() int {
	return len(sorter.list)
}

// Swap swaps the elements of underlying WorkerInfo slice. It is part of sort.Interface.
func (sorter *sorter) Swap(i, j int) {
	sorter.list[i], sorter.list[j] = sorter.list[j], sorter.list[i]
}

// Less reports if element with index i in underlying WorkerInfo slice should be before the element
// with index j. It may panic when item wasn't set or was set to wrong value. It is part of
// sort.Interface.
func (sorter *sorter) Less(i, j int) bool {
	var less func(a, b int) bool

	applySortOrder := func(retval bool) bool {
		if sorter.order == boruta.SortOrderAsc {
			return retval
		}
		return !retval
	}

	lessUUID := func(a, b int) bool {
		return applySortOrder(sorter.list[a].WorkerUUID < sorter.list[b].WorkerUUID)
	}

	switch sorter.item {
	case "":
		// UUID is the default sort item.
		fallthrough
	case "uuid":
		less = lessUUID
	case "state":
		less = func(a, b int) bool {
			if sorter.list[a].State == sorter.list[b].State {
				return lessUUID(a, b)
			}
			return applySortOrder(sorter.list[a].State < sorter.list[b].State)
		}
	case "groups":
		less = func(a, b int) bool {
			var min int
			ga := sorter.list[a].Groups
			gb := sorter.list[b].Groups
			la := len(ga)
			lb := len(gb)
			if la <= lb {
				min = la
			} else {
				min = lb
			}
			for i := 0; i < min; i++ {
				if ga[i] < gb[i] {
					return applySortOrder(true)
				} else if ga[i] > gb[i] {
					return applySortOrder(false)
				}
			}
			// Up to min groups are equal.
			if la == lb {
				return lessUUID(a, b)
			}
			return applySortOrder(la < lb)
		}
	default:
		panic(sorter.item + ": " + boruta.ErrWrongSortItem.Error())
	}
	return less(i, j)
}
