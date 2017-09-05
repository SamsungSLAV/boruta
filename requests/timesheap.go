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

// File requests/timesheap.go provides minimum heap implementation for requestTime.
// timesHeap structure provides easy API for usage in higher layers of requests
// package. It uses timesHeapContainer for heap implementation.

package requests

import (
	"container/heap"
)

// timesHeap implements requestTime heap. Its methods build API easy to be used
// by higher layers of requests package.
type timesHeap struct {
	con timesHeapContainer
}

// newTimesHeap creates new timesHeap object and initializes it as an empty heap.
func newTimesHeap() *timesHeap {
	h := new(timesHeap)
	heap.Init(&h.con)
	return h
}

// Len returns size of the heap.
func (h *timesHeap) Len() int {
	return h.con.Len()
}

// Min returns the minimum heap element (associated with the earliest time).
// The returned element is not removed from the heap. Size of the heap doesn't
// change. It panics if the heap is empty.
func (h *timesHeap) Min() requestTime {
	if h.con.Len() == 0 {
		panic("cannot get Min of empty heap")
	}
	return h.con[0]
}

// Pop removes and returns the minimum heap element that is associated with
// earliest time. Heap's size is decreased by one.
// Method panics if the heap is empty.
func (h *timesHeap) Pop() requestTime {
	return heap.Pop(&h.con).(requestTime)
}

// Push adds the requestTime element to the heap, keeping its minimum value
// property. Heap's size is increased by one.
func (h *timesHeap) Push(t requestTime) {
	heap.Push(&h.con, t)
}
