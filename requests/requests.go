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

// Package requests provides structures and functions to handle requests.
package requests

import (
	"sync"
	"time"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/matcher"
)

// ReqsCollection contains information (also historical) about handled requests.
// It implements Requests and RequestsManager interfaces.
type ReqsCollection struct {
	requests          map[ReqID]*ReqInfo
	queue             *prioQueue
	mutex             *sync.RWMutex
	iterating         bool
	workers           matcher.WorkersManager
	jobs              matcher.JobsManager
	validAfterTimes   *requestTimes
	deadlineTimes     *requestTimes
	timeoutTimes      *requestTimes
	validAfterMatcher matcher.Matcher
	deadlineMatcher   matcher.Matcher
	timeoutMatcher    matcher.Matcher
}

// NewRequestQueue provides initialized priority queue for requests.
func NewRequestQueue(w matcher.WorkersManager, j matcher.JobsManager) *ReqsCollection {
	r := &ReqsCollection{
		requests:        make(map[ReqID]*ReqInfo),
		queue:           newPrioQueue(),
		mutex:           new(sync.RWMutex),
		workers:         w,
		jobs:            j,
		validAfterTimes: newRequestTimes(),
		deadlineTimes:   newRequestTimes(),
		timeoutTimes:    newRequestTimes(),
	}

	r.validAfterMatcher = matcher.NewValidMatcher(r, w, j)
	r.deadlineMatcher = matcher.NewDeadlineMatcher(r)
	r.timeoutMatcher = matcher.NewTimeoutMatcher(r)

	r.validAfterTimes.setMatcher(r.validAfterMatcher)
	r.deadlineTimes.setMatcher(r.deadlineMatcher)
	r.timeoutTimes.setMatcher(r.timeoutMatcher)

	return r
}

// Finish releases requestTimes queues and stops started goroutines.
func (reqs *ReqsCollection) Finish() {
	reqs.validAfterTimes.finish()
	reqs.deadlineTimes.finish()
	reqs.timeoutTimes.finish()
}

// NewRequest is part of implementation of Requests interface. It validates
// provided arguments and creates request or returns an error. Caller must make
// sure that provided time values are in UTC.
func (reqs *ReqsCollection) NewRequest(caps Capabilities,
	priority Priority, owner UserInfo, validAfter time.Time,
	deadline time.Time) (ReqID, error) {

	req := &ReqInfo{
		ID:         ReqID(len(reqs.requests) + 1),
		Priority:   priority,
		Owner:      owner,
		Deadline:   deadline,
		ValidAfter: validAfter,
		State:      WAIT,
		Caps:       caps,
	}

	if !req.Deadline.IsZero() && req.Deadline.Before(time.Now().UTC()) {
		return 0, ErrDeadlineInThePast
	}

	if req.ValidAfter.After(req.Deadline) && !req.Deadline.IsZero() {
		return 0, ErrInvalidTimeRange
	}

	if req.ValidAfter.IsZero() {
		req.ValidAfter = time.Now().UTC()
	}
	if req.Deadline.IsZero() {
		// TODO(mwereski): make defaultValue configurable when config is
		// introduced
		req.Deadline = time.Now().AddDate(0, 1, 0).UTC()
	}

	// TODO(mwereski): Check if user has rights to set given priority.
	if req.Priority < HiPrio || req.Priority > LoPrio {
		return 0, ErrPriority
	}

	// TODO(mwereski): Check if capabilities can be satisfied.

	reqs.mutex.Lock()
	reqs.queue.pushRequest(req)
	reqs.requests[req.ID] = req
	reqs.mutex.Unlock()

	reqs.validAfterTimes.insert(requestTime{time: req.ValidAfter, req: req.ID})
	reqs.deadlineTimes.insert(requestTime{time: req.Deadline, req: req.ID})

	return req.ID, nil
}

// closeRequest is an internal ReqsCollection method for closing running request.
// It is used by both Close and CloseRequest methods after verification that
// all required conditions to close request are met.
// The method must be called in reqs.mutex critical section.
func (reqs *ReqsCollection) closeRequest(req *ReqInfo) {
	req.State = DONE
	// TODO(mwereski): release worker
}

// CloseRequest is part of implementation of Requests interface.
// It checks that request is in WAIT state and changes it to CANCEL or
// in INPROGRESS state and changes it to DONE. NotFoundError may be returned
// if request with given reqID doesn't exist in the queue
// or ErrModificationForbidden if request is in state which can't be closed.
func (reqs *ReqsCollection) CloseRequest(reqID ReqID) error {
	reqs.mutex.Lock()
	defer reqs.mutex.Unlock()
	req, ok := reqs.requests[reqID]
	if !ok {
		return NotFoundError("Request")
	}
	switch req.State {
	case WAIT:
		req.State = CANCEL
		reqs.queue.removeRequest(req)
	case INPROGRESS:
		reqs.closeRequest(req)
	default:
		return ErrModificationForbidden
	}
	return nil
}

// modificationPossible is simple helper function that checks if it is possible
// to modify request it given state.
func modificationPossible(state ReqState) bool {
	return state == WAIT
}

// UpdateRequest is part of implementation of Requests interface. It may be used
// to modify ValidAfter, Deadline or Priority of request. Caller should pass
// pointer to new ReqInfo struct which has any of these fields set. Zero value
// means that field shouldn't be changed. All fields that cannot be changed are
// ignored.
func (reqs *ReqsCollection) UpdateRequest(src *ReqInfo) error {
	if src == nil || (src.Priority == Priority(0) &&
		src.ValidAfter.IsZero() &&
		src.Deadline.IsZero()) {
		return nil
	}
	validAfterTime, deadlineTime, err := reqs.updateRequest(src)
	if err != nil {
		return err
	}
	if validAfterTime != nil {
		reqs.validAfterTimes.insert(*validAfterTime)
	}
	if deadlineTime != nil {
		reqs.deadlineTimes.insert(*deadlineTime)
	}
	return nil
}

// updateRequest is a part of UpdateRequest implementation run in critical section.
func (reqs *ReqsCollection) updateRequest(src *ReqInfo) (validAfterTime, deadlineTime *requestTime, err error) {
	reqs.mutex.Lock()
	defer reqs.mutex.Unlock()

	dst, ok := reqs.requests[src.ID]
	if !ok {
		err = NotFoundError("Request")
		return
	}
	if !modificationPossible(dst.State) {
		err = ErrModificationForbidden
		return
	}
	if src.Priority == dst.Priority &&
		src.ValidAfter.Equal(dst.ValidAfter) &&
		src.Deadline.Equal(dst.Deadline) {
		return
	}
	// TODO(mwereski): Check if user has rights to set given priority.
	if src.Priority != Priority(0) && (src.Priority < HiPrio ||
		src.Priority > LoPrio) {
		err = ErrPriority
		return
	}
	deadline := dst.Deadline
	if !src.Deadline.IsZero() {
		if src.Deadline.Before(time.Now().UTC()) {
			err = ErrDeadlineInThePast
			return
		}
		deadline = src.Deadline
	}
	if (!src.ValidAfter.IsZero()) && !deadline.IsZero() &&
		src.ValidAfter.After(deadline) {
		err = ErrInvalidTimeRange
		return
	}

	if src.Priority != Priority(0) {
		reqs.queue.setRequestPriority(dst, src.Priority)
		dst.Priority = src.Priority
	}
	if !src.ValidAfter.IsZero() {
		dst.ValidAfter = src.ValidAfter
		validAfterTime = &requestTime{time: src.ValidAfter, req: src.ID}
	}
	if !dst.Deadline.Equal(deadline) {
		dst.Deadline = deadline
		deadlineTime = &requestTime{time: deadline, req: src.ID}
	}
	return
}

// GetRequestInfo is part of implementation of Requests interface. It returns
// ReqInfo for given request ID or NotFoundError if request with given ID doesn't
// exits in the collection.
func (reqs *ReqsCollection) GetRequestInfo(reqID ReqID) (ReqInfo, error) {
	reqs.mutex.RLock()
	defer reqs.mutex.RUnlock()
	return reqs.Get(reqID)
}

// ListRequests is part of implementation of Requests interface. It returns slice
// of ReqInfo that matches ListFilter.
func (reqs *ReqsCollection) ListRequests(filter ListFilter) ([]ReqInfo, error) {
	reqs.mutex.RLock()
	defer reqs.mutex.RUnlock()
	res := make([]ReqInfo, 0, len(reqs.requests))
	for _, req := range reqs.requests {
		if filter == nil || filter.Match(req) {
			res = append(res, *req)
		}
	}
	return res, nil
}

// AcquireWorker is part of implementation of Requests interface. When worker is
// assigned to the requests then owner of such requests may call AcquireWorker
// to get all information required to use assigned worker.
func (reqs *ReqsCollection) AcquireWorker(reqID ReqID) (AccessInfo, error) {
	reqs.mutex.RLock()
	defer reqs.mutex.RUnlock()
	req, ok := reqs.requests[reqID]
	if !ok {
		return AccessInfo{}, NotFoundError("Request")
	}
	if req.State != INPROGRESS || req.Job == nil {
		return AccessInfo{}, ErrWorkerNotAssigned
	}
	// TODO(mwereski): create job and get access info
	return AccessInfo{}, nil
}

// ProlongAccess is part of implementation of Requests interface. When owner of
// the request has acquired worker that to extend time for which the worker is
// assigned to the request.
func (reqs *ReqsCollection) ProlongAccess(reqID ReqID) error {
	reqs.mutex.RLock()
	defer reqs.mutex.RUnlock()
	req, ok := reqs.requests[reqID]
	if !ok {
		return NotFoundError("Request")
	}
	if req.State != INPROGRESS || req.Job == nil {
		return ErrWorkerNotAssigned
	}
	// TODO(mwereski): prolong access
	return nil
}

// InitIteration initializes queue iterator and sets global lock for requests
// structures. It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) InitIteration() error {
	reqs.mutex.Lock()
	if reqs.iterating {
		reqs.mutex.Unlock()
		return ErrInternalLogicError
	}
	reqs.queue.initIterator()
	reqs.iterating = true
	return nil
}

// TerminateIteration releases queue iterator if iterations are in progress
// and release global lock for requests structures.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) TerminateIteration() {
	if reqs.iterating {
		reqs.queue.releaseIterator()
		reqs.iterating = false
	}
	reqs.mutex.Unlock()
}

// Next gets next ID from request queue. Method returns {ID, true} if there is
// pending request or {ReqID(0), false} if queue's end has been reached.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) Next() (ReqID, bool) {
	if reqs.iterating {
		return reqs.queue.next()
	}
	panic("Should never call Next(), when not iterating")
}

// VerifyIfReady checks if the request is ready to be run on worker.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) VerifyIfReady(rid ReqID, now time.Time) bool {
	req, ok := reqs.requests[rid]
	return ok && req.State == WAIT && req.Deadline.After(now) && !req.ValidAfter.After(now)
}

// Get retrieves request's information structure for request with given ID.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) Get(rid ReqID) (ReqInfo, error) {
	req, ok := reqs.requests[rid]
	if !ok {
		return ReqInfo{}, NotFoundError("Request")
	}
	return *req, nil
}

// Timeout sets request to TIMEOUT state after Deadline time is exceeded.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) Timeout(rid ReqID) error {
	reqs.mutex.Lock()
	defer reqs.mutex.Unlock()
	req, ok := reqs.requests[rid]
	if !ok {
		return NotFoundError("Request")
	}
	if req.State != WAIT || req.Deadline.After(time.Now()) {
		return ErrModificationForbidden
	}
	req.State = TIMEOUT
	reqs.queue.removeRequest(req)
	return nil
}

// Close verifies if request time has been exceeded and if so closes it.
// If request is still valid to continue it's job an error is returned.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) Close(reqID ReqID) error {
	reqs.mutex.Lock()
	defer reqs.mutex.Unlock()
	req, ok := reqs.requests[reqID]
	if !ok {
		return NotFoundError("Request")
	}
	if req.State != INPROGRESS {
		return ErrModificationForbidden
	}
	if req.Job == nil {
		// TODO log a critical logic error. Job should be assigned to the request
		// in INPROGRESS state.
		return ErrInternalLogicError
	}
	if req.Job.Timeout.After(time.Now()) {
		// Request prolonged not yet ready to be closed because of timeout.
		return ErrModificationForbidden
	}

	reqs.closeRequest(req)

	return nil
}

// Run starts job performing the request on the worker.
// It is part of implementation of RequestsManager interface.
func (reqs *ReqsCollection) Run(rid ReqID, worker WorkerUUID) error {
	req, ok := reqs.requests[rid]
	if !ok {
		return NotFoundError("Request")
	}

	if req.State != WAIT {
		return ErrModificationForbidden
	}
	req.State = INPROGRESS

	if reqs.iterating {
		reqs.queue.releaseIterator()
		reqs.iterating = false
	}
	reqs.queue.removeRequest(req)

	// TODO(lwojciechow) assign req.Job.
	return nil
}
