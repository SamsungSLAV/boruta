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

// File requests/timesheapcontainer.go provides implementation of heap.Interface
// on requestTime slice. Structure is used for organizing requests' times
// into the heap with minimum property.

package requests

import (
	"time"

	. "git.tizen.org/tools/boruta"
)

// requestTime combines ReqID with time.Time.
type requestTime struct {
	time time.Time // stores point in time.
	req  ReqID     // identifies request related to the time.
}

// timesHeapContainer wraps requestTime slice for implementation of heap.Interface.
type timesHeapContainer []requestTime

// Len returns current heap size. It is a part of container.heap.Interface
// implementation by timesHeapContainer.
func (h timesHeapContainer) Len() int {
	return len(h)
}

// Less compares 2 requestTime elements in the heap by time and then by ReqID
// It is a part of container.heap.Interface implementation by timesHeapContainer.
func (h timesHeapContainer) Less(i, j int) bool {
	if h[i].time.Equal(h[j].time) {
		return h[i].req < h[j].req
	}
	return h[i].time.Before(h[j].time)
}

// Swap exchanges 2 heap elements one with other.
// It is a part of container.heap.Interface implementation by timesHeapContainer.
func (h timesHeapContainer) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

// Push adds an element to the heap.
// It is a part of container.heap.Interface implementation by timesHeapContainer.
func (h *timesHeapContainer) Push(x interface{}) {
	*h = append(*h, x.(requestTime))
}

// Pop removes and returns last element of the heap.
// It is a part of container.heap.Interface implementation by timesHeapContainer.
func (h *timesHeapContainer) Pop() interface{} {
	n := h.Len()
	if n == 0 {
		panic("cannot Pop from empty heap")
	}

	x := (*h)[n-1]
	*h = (*h)[:n-1]

	return x
}
