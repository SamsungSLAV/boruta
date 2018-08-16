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

// File lists.go contains all data structures that are used in listing collections of requests and
// workers.

package boruta

import "strings"

// ListFilter is used to filter Requests in the Queue.
type ListFilter interface {
	// Match tells if request matches the filter.
	Match(req *ReqInfo) bool
}

// SortOrder denotes in which order (ascending or descending) collection should be sorted.
type SortOrder bool

const (
	// SortOrderAsc means ascending order. This is the default.
	SortOrderAsc SortOrder = false
	// SortOrderDesc means descending order.
	SortOrderDesc SortOrder = true
)

// SortInfo contains information needed to sort collections in Boruta (requests, workers).
type SortInfo struct {
	// Item by which collection should be sorted.
	Item string
	// Order in which collection should be sorted.
	Order SortOrder
}

// String returns textual representation of SortOrder ("ascending" or "descending").
func (order SortOrder) String() string {
	if order == SortOrderDesc {
		return "descending"
	}
	return "ascending"
}

// MarshalText is implementation of encoding.TextMarshaler interface. It is used to properly
// marshal from structures that contain SortOrder members.
func (order *SortOrder) MarshalText() ([]byte, error) {
	return []byte(order.String()), nil
}

// UnmarshalText is implementation of encoding.TextUnmarshaler interface. It is used to properly
// unmarshal structures that contain SortOrder members.
func (order *SortOrder) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "":
		fallthrough // ascending is the default order
	case SortOrderAsc.String():
		*order = SortOrderAsc
		return nil
	case SortOrderDesc.String():
		*order = SortOrderDesc
		return nil
	default:
		return ErrWrongSortOrder
	}
}
