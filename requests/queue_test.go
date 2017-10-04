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

// File requests/queue_test.go contains additional tests for queue.go. Please
// take a look at requests_test.go for initTest() and requestsTests definition.

package requests

import (
	"testing"

	. "git.tizen.org/tools/boruta"
)

func TestRemovePanic(t *testing.T) {
	assert, rqueue := initTest(t)
	assert.Panics(func() { rqueue.queue._remove(ReqID(1), LoPrio) })
}

func TestQueue(t *testing.T) {
	assert, rqueue := initTest(t)
	var reqs = []struct {
		id ReqID
		pr Priority
	}{
		{ReqID(1), Priority(7)},
		{ReqID(2), Priority(1)},
		{ReqID(3), Priority(2)},
		{ReqID(4), Priority(12)},
		{ReqID(5), Priority(3)},
		{ReqID(6), Priority(3)},
	}
	sorted := []ReqID{ReqID(2), ReqID(3), ReqID(5), ReqID(6), ReqID(1), ReqID(4)}

	// Test for empty queue.
	reqid, ok := rqueue.queue.next()
	assert.False(ok)
	assert.Equal(ReqID(0), reqid)

	// Test if iterator was initialized and queue is empty.
	rqueue.queue.initIterator()
	reqid, ok = rqueue.queue.next()
	assert.False(ok)
	assert.Equal(ReqID(0), reqid)
	rqueue.queue.releaseIterator()

	req := requestsTests[0].req
	// Push requests to the queue.
	for _, r := range reqs {
		_, err := rqueue.NewRequest(req.Caps, r.pr, req.Owner, req.ValidAfter, req.Deadline)
		assert.Nil(err)
	}

	// Check if queue returns request IDs in proper order.
	rqueue.queue.initIterator()
	for _, r := range sorted {
		reqid, ok = rqueue.queue.next()
		assert.True(ok)
		assert.Equal(r, reqid)
	}

	// Check if call to next() after iterating through whole queue returns false.
	reqid, ok = rqueue.queue.next()
	assert.False(ok)
	assert.Equal(ReqID(0), reqid)
	rqueue.queue.releaseIterator()

	// Check if after another initialization next() returns first element.
	rqueue.queue.initIterator()
	reqid, ok = rqueue.queue.next()
	assert.True(ok)
	assert.Equal(sorted[0], reqid)
	// Check call to releaseIterator() when iterator hasn't finished properly
	// sets next().
	rqueue.queue.releaseIterator()
	reqid, ok = rqueue.queue.next()
	assert.False(ok)
	assert.Equal(ReqID(0), reqid)
}
