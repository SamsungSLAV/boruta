/*
 *  Copyright (c) 2017 Samsung Electronics Co., Ltd All Rights Reserved
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

// File requests/queue.go file contains implementation of Priority Queue for
// requests. It's done as an array of regular FIFO queues - one per priority.

package requests

import (
	"container/list"
	"sync"

	. "git.tizen.org/tools/boruta"
)

// prioQueue is priority queue that stores request IDs.
// Following part of interface should be used:
// - pushRequest()
// - removeRequest()
// - setRequestPriority()
// - initIterator()
// - releaseIterator()
// - next()
type prioQueue struct {
	queue  []*list.List
	length uint
	// next returns ID of next request in the priority queue and bool which
	// indicates if ID was found. False means that caller has iterated through
	// all elements and pq.releaseIterator() followed by pq.initIterator()
	// must be called in order to have a working iterator again.
	next func() (ReqID, bool)
	mtx  *sync.Mutex
}

// _emptyIterator is helper function which always returns values which indicate
// that iterator should be initialized. It is desired to be set as next member of
// prioQueue structure whenever iterator needs initialization.
func _emptyIterator() (ReqID, bool) { return ReqID(0), false }

// newPrioQueue returns pointer to newly created and initialized priority queue.
func newPrioQueue() *prioQueue {
	pq := new(prioQueue)

	// Prepare queues.
	pq.queue = make([]*list.List, LoPrio+1)
	for i := HiPrio; i <= LoPrio; i++ {
		pq.queue[i] = new(list.List).Init()
	}
	pq.length = 0
	pq.mtx = new(sync.Mutex)

	// Prepare iterator.
	pq.next = _emptyIterator

	return pq
}

// _remove removes request with given reqID from the queue. Caller must be sure
// that request with given ID exists in the queue otherwise function will panic.
// It's more convenient to use removeRequest().
func (pq *prioQueue) _remove(reqID ReqID, priority Priority) {
	for e := pq.queue[priority].Front(); e != nil; e = e.Next() {
		if e.Value.(ReqID) == reqID {
			pq.length--
			pq.queue[priority].Remove(e)
			return
		}
	}
	panic("request with given reqID doesn't exist in the queue")
}

// removeRequest removes request from the priority queue. It wraps _remove(),
// which will panic if request is missing from the queue and removeRequest will
// propagate this panic.
func (pq *prioQueue) removeRequest(req *ReqInfo) {
	pq.mtx.Lock()
	defer pq.mtx.Unlock()
	pq._remove(req.ID, req.Priority)
}

// _push adds request ID at the end of priority queue. It's more convenient to use
// pushRequest().
func (pq *prioQueue) _push(reqID ReqID, priority Priority) {
	pq.queue[priority].PushBack(reqID)
	pq.length++
}

// pushRequest adds request to priority queue. It wraps _push().
func (pq *prioQueue) pushRequest(req *ReqInfo) {
	pq.mtx.Lock()
	defer pq.mtx.Unlock()
	pq._push(req.ID, req.Priority)
}

// setRequestPriority modifies priority of request that was already added to the
// queue. Caller must make sure that request with given ID exists in the queue.
// Panic will occur if such ID doesn't exist.
func (pq *prioQueue) setRequestPriority(req *ReqInfo, newPrio Priority) {
	pq.mtx.Lock()
	defer pq.mtx.Unlock()
	pq._remove(req.ID, req.Priority)
	pq._push(req.ID, newPrio)
}

// initIterator initializes iterator. Caller must call it before first call to pq.next().
// If caller wants to iterate once again through the queue (e.g. after pq.next()
// returns (0, false)), then (s)he must call pq.releaseIterator and initIterator()
// once again.
func (pq *prioQueue) initIterator() {
	pq.mtx.Lock()
	// current priority
	p := HiPrio
	// current element of list for p priority
	e := pq.queue[p].Front()

	pq.next = func() (id ReqID, ok bool) {

		// The queue is empty.
		if pq.length == 0 {
			p = HiPrio
			e = nil
			return ReqID(0), false
		}

		if e == nil {
			// Find next priority.
			for p++; p <= LoPrio && pq.queue[p].Len() == 0; p++ {
			}
			if p > LoPrio {
				return ReqID(0), false
			}
			// Get it's first element.
			e = pq.queue[p].Front()
		}

		id, ok = e.Value.(ReqID), true
		e = e.Next()
		return
	}
}

// releaseIterator must be called after user finishes iterating through the queue.
func (pq *prioQueue) releaseIterator() {
	pq.next = _emptyIterator
	pq.mtx.Unlock()
}
