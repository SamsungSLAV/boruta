/*
 *  Copyright (c) 2017-2022 Samsung Electronics Co., Ltd All Rights Reserved
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

// File requests/queue_test.go contains additional tests for queue.go. Please
// take a look at requests_test.go for requestsTests definition.

package requests

import (
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

func TestRemovePanic(t *testing.T) {
	assert := assert.New(t)
	queue := newPrioQueue()
	assert.Panics(func() { queue._remove(boruta.ReqID(1), boruta.LoPrio) })
}

func TestQueue(t *testing.T) {
	assert := assert.New(t)
	queue := newPrioQueue()

	runTest := func(exists bool, expected boruta.ReqID) {
		t.Helper()
		reqid, ok := queue.next()
		assert.Equal(exists, ok)
		assert.Equal(expected, reqid)
	}

	reqs := [...]struct {
		id boruta.ReqID
		pr boruta.Priority
	}{
		{boruta.ReqID(1), boruta.Priority(7)},
		{boruta.ReqID(2), boruta.Priority(1)},
		{boruta.ReqID(3), boruta.Priority(2)},
		{boruta.ReqID(4), boruta.Priority(12)},
		{boruta.ReqID(5), boruta.Priority(3)},
		{boruta.ReqID(6), boruta.Priority(3)},
	}
	sorted := [...]boruta.ReqID{boruta.ReqID(2), boruta.ReqID(3), boruta.ReqID(5), boruta.ReqID(6),
		boruta.ReqID(1), boruta.ReqID(4)}

	// Test for empty queue.
	runTest(false, boruta.ReqID(0))

	// Test if iterator was initialized and queue is empty.
	queue.initIterator()
	runTest(false, boruta.ReqID(0))
	queue.releaseIterator()

	req := requestsTests[0].req
	// Push requests to the queue.
	for _, r := range reqs {
		queue.pushRequest(&boruta.ReqInfo{
			ID:         r.id,
			Priority:   r.pr,
			Owner:      req.Owner,
			Deadline:   req.Deadline,
			ValidAfter: req.ValidAfter,
			State:      boruta.WAIT,
			Caps:       req.Caps,
		})
	}

	// Check if queue returns request IDs in proper order.
	queue.initIterator()
	for _, r := range sorted {
		runTest(true, r)
	}

	// Check if call to next() after iterating through whole queue returns false.
	runTest(false, boruta.ReqID(0))
	queue.releaseIterator()

	// Check if after another initialization next() returns first element.
	queue.initIterator()
	runTest(true, sorted[0])
	// Check call to releaseIterator() when iterator hasn't finished properly
	// sets next().
	queue.releaseIterator()
	runTest(false, boruta.ReqID(0))
}
