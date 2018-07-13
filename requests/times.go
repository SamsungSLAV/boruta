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

// File requests/times.go provides requestTimes structure, which collects
// time.Time objects associated with requests and notifies registered
// matcher.Matcher when proper time comes. Past times are removed from collection.
// timesHeap is used for storing time.Time objects. Notifications are called
// asynchronously from dedicated goroutine.

package requests

import (
	"sync"
	"time"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/matcher"
)

// requestTimes collects requestTime entities and notifies registered
// matcher.Matcher when the time comes.
type requestTimes struct {
	times   *timesHeap      // stores requestTime entities collection.
	timer   *time.Timer     // set for earliest time in the collection.
	matcher matcher.Matcher // notified when time is reached.
	mutex   *sync.Mutex     // synchronizes internal goroutine.
	stop    chan bool       // stops internal goroutine.
	done    sync.WaitGroup  // waits for internal goroutine to finish.
}

// newRequestTimes creates and initializes new requestTimes structure.
// It runs internal goroutine. finish() method should be used for clearing object
// and stopping internal goroutine.
func newRequestTimes() *requestTimes {
	farFuture := time.Now().AddDate(100, 0, 0)
	rt := &requestTimes{
		times: newTimesHeap(),
		timer: time.NewTimer(time.Until(farFuture)),
		mutex: new(sync.Mutex),
		stop:  make(chan bool),
	}
	rt.done.Add(1)
	go rt.loop()
	return rt
}

// finish clears requestTimes object and stops internal goroutine. It should be
// called exactly once. Object cannot be used after calling this method anymore.
func (rt *requestTimes) finish() {
	// Stop timer.
	rt.timer.Stop()

	// Break loop goroutine and wait until it's done.
	close(rt.stop)
	rt.done.Wait()
}

// loop is the main procedure of the internal goroutine. It waits for either timer
// event or being stopped by Finish.
func (rt *requestTimes) loop() {
	defer rt.done.Done()
	for {
		// get event from timer
		select {
		case t := <-rt.timer.C:
			rt.process(t)
		case <-rt.stop:
			return
		}
	}
}

// process notifies registered matcher, removes all past times from the collection
// and sets up timer for earliest time from the collection.
func (rt *requestTimes) process(t time.Time) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	// Remove all past times. It might happen that the same point in time is
	// added multiple times or that deltas between added values are so small,
	// that in the time of processing some of next pending points in time
	// are already in the past. There is no need to set timer for them,
	// as it will return immediately. So all past times must be removed
	// and timer set to earliest future time.
	past := make([]boruta.ReqID, 0)
	for rt.times.Len() > 0 && t.After(rt.minTime()) {
		x := rt.times.Pop()
		past = append(past, x.req)
	}

	// Notify matcher (if one is registered).
	if rt.matcher != nil {
		rt.matcher.Notify(past)
	}

	// Set up timer to earliest pending time.
	if rt.times.Len() > 0 {
		rt.reset()
	}
}

// insert adds time to the collection and possibly restarts timer if required.
func (rt *requestTimes) insert(t requestTime) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.times.Push(t)

	// If inserted time is minimal (first element of the heap) timer needs to be
	// restarted.
	if rt.minTime().Equal(t.time) {
		rt.timer.Stop()
		rt.reset()
	}
}

// setMatcher registers object implementing Matcher interface. Registered object
// is notified, when collected times pass.
func (rt *requestTimes) setMatcher(m matcher.Matcher) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	rt.matcher = m
}

// minTime is a helper function extracting minimal time from the heap. This method
// should be called in requestTimes mutex protected critical section.
// Method panics if called on empty collection.
func (rt *requestTimes) minTime() time.Time {
	return rt.times.Min().time
}

// reset is the helper function for resetting stopped timer to the time of next event.
// This method should be called in requestTimes mutex protected critical section.
func (rt *requestTimes) reset() {
	rt.timer.Reset(time.Until(rt.minTime()))
}
