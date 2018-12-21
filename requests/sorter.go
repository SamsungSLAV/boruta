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

// File requests/sorter.go provides implementation of sorter. It is used by ListRequests().

package requests

import "github.com/SamsungSLAV/boruta"

// sorter implements sort.Interface. It allows to sort by request ID (default), Priority,
// Deadline, ValidAfter or State. It also contains order information (ascending or descending).
// It is used by ListRequests.
type sorter struct {
	reqs  []boruta.ReqInfo
	item  string
	order boruta.SortOrder
}

// newSorter returns sorter initialized with SortInfo data. It returns an error when info is nil or
// it contains unknown item.
func newSorter(info *boruta.SortInfo) (*sorter, error) {
	if info == nil {
		info = new(boruta.SortInfo)
	}
	switch info.Item {
	case "":
	case "id":
	case "priority":
	case "deadline":
	case "validafter":
	case "state":
	default:
		return nil, boruta.ErrWrongSortItem
	}
	return &sorter{
		order: info.Order,
		item:  info.Item,
	}, nil
}

// Len returns length of slice of underlying ReqInfo slice. It is part of sort.Interface.
func (sorter *sorter) Len() int {
	return len(sorter.reqs)
}

// Swap swaps the elements of underlying ReqInfo slice. It is part of sort.Interface.
func (sorter *sorter) Swap(i, j int) {
	sorter.reqs[i], sorter.reqs[j] = sorter.reqs[j], sorter.reqs[i]
}

// Less reports if element with index i in underlying ReqInfo slice should be before the element
// with index j. It may panic when item wasn't set or was set to wrong value. It is part of
// sort.Interface.
func (sorter *sorter) Less(i, j int) bool {
	var less func(a, b int) bool
	lessID := func(a, b int) bool {
		if sorter.order == boruta.SortOrderAsc {
			return sorter.reqs[a].ID < sorter.reqs[b].ID
		}
		return sorter.reqs[b].ID < sorter.reqs[a].ID
	}
	switch sorter.item {
	case "":
		// ID is the default sort item.
		fallthrough
	case "id":
		less = lessID
	case "priority":
		less = func(a, b int) bool {
			if sorter.reqs[a].Priority == sorter.reqs[b].Priority {
				return lessID(a, b)
			}
			if sorter.order == boruta.SortOrderAsc {
				return sorter.reqs[a].Priority < sorter.reqs[b].Priority
			}
			return sorter.reqs[b].Priority < sorter.reqs[a].Priority
		}
	case "deadline":
		less = func(a, b int) bool {
			if sorter.reqs[a].Deadline.Equal(sorter.reqs[b].Deadline) {
				return lessID(a, b)
			}
			if sorter.order == boruta.SortOrderAsc {
				return sorter.reqs[a].Deadline.Before(sorter.reqs[b].Deadline)
			}
			return sorter.reqs[b].Deadline.Before(sorter.reqs[a].Deadline)
		}
	case "validafter":
		less = func(a, b int) bool {
			if sorter.reqs[a].ValidAfter.Equal(sorter.reqs[b].ValidAfter) {
				return lessID(a, b)
			}
			if sorter.order == boruta.SortOrderAsc {
				return sorter.reqs[a].ValidAfter.Before(sorter.reqs[b].ValidAfter)
			}
			return sorter.reqs[b].ValidAfter.Before(sorter.reqs[a].ValidAfter)
		}
	case "state":
		less = func(a, b int) bool {
			if sorter.reqs[a].State == sorter.reqs[b].State {
				return lessID(a, b)
			}
			if sorter.order == boruta.SortOrderAsc {
				return sorter.reqs[a].State < sorter.reqs[b].State
			}
			return sorter.reqs[b].State < sorter.reqs[a].State
		}
	default:
		panic(sorter.item + ": " + boruta.ErrWrongSortItem.Error())
	}
	return less(i, j)
}
