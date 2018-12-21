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

import (
	"math"
	"strings"
)

// ListFilter is used to filter elements in a collection.
type ListFilter interface {
	// Match tells if element matches the filter.
	Match(elem interface{}) bool
}

// SortOrder denotes in which order (ascending or descending) collection should be sorted.
type SortOrder bool

// ListDirection denotes in which direction collection should be traversed.
type ListDirection bool

const (
	// SortOrderAsc means ascending order. This is the default.
	SortOrderAsc SortOrder = false
	// SortOrderDesc means descending order.
	SortOrderDesc SortOrder = true
	// DirectionForward means that list is traversed in forward direction. This is the default.
	DirectionForward ListDirection = false
	// DirectionBackward means that list is traversed in backward direction.
	DirectionBackward ListDirection = true

	// MaxPageLimit denotes maximum value that pageInfo.Limit can be.
	MaxPageLimit = math.MaxUint16
)

// SortInfo contains information needed to sort collections in Boruta (requests, workers).
type SortInfo struct {
	// Item by which collection should be sorted.
	Item string
	// Order in which collection should be sorted.
	Order SortOrder
}

// ListInfo contains information about filtered list - how many items are there and how many items
// are left till the end.
type ListInfo struct {
	// TotalItems contains information how many items in total is in filtered collection.
	TotalItems uint64
	// RemainingItems contains information how many items are left till the end of colleciton
	// (when paginating).
	RemainingItems uint64
}

// RequestsPaginator contains information to get specific page of listed requests.
type RequestsPaginator struct {
	// ID sets page border. When direction is set to forward the page will start with first
	// request after the ID and contain up to Limit items. If direction is set backward then
	// page contains up to Limit items before ID.
	ID ReqID
	// Direction in which list should be traversed.
	Direction ListDirection
	// Limit up to how many elements can be stored on one page.
	Limit uint16
}

// WorkersPaginator contains information to get specific page of listed workers.
type WorkersPaginator struct {
	// ID sets page border. When direction is set to forward the page will start with first
	// workes after the ID and contain up to Limit items. If direction is set backward then
	// page contains up to Limit items before ID.
	ID WorkerUUID
	// Direction in which list should be traversed.
	Direction ListDirection
	// Limit up to how many elements can be stored on one page.
	Limit uint16
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

// String returns textual representation of ListDirection ("forward" or "backward").
func (direction ListDirection) String() string {
	if direction == DirectionBackward {
		return "backward"
	}
	return "forward"
}

// MarshalText is implementation of encoding.TextMarshaler interface. It is used to properly
// marshal from structures that contain ListDirection members.
func (direction *ListDirection) MarshalText() ([]byte, error) {
	return []byte(direction.String()), nil
}

// UnmarshalText is implementation of encoding.TextUnmarshaler interface. It is used to properly
// unmarshal structures that contain ListDirection members.
func (direction *ListDirection) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case "":
		fallthrough // forward is the default direction
	case DirectionForward.String():
		*direction = DirectionForward
	case DirectionBackward.String():
		*direction = DirectionBackward
	default:
		return ErrWrongListDirection
	}
	return nil
}
