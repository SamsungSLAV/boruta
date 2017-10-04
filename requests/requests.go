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

// Package requests provides structures and functions to handle requests.
package requests

import (
	"sync"
	"time"

	. "git.tizen.org/tools/boruta"
)

// ReqsCollection contains information (also historical) about handled requests.
// It implements Requests interface.
type ReqsCollection struct {
	requests map[ReqID]*ReqInfo
	queue    *prioQueue
	mutex    *sync.RWMutex
}

// NewRequestQueue provides initialized priority queue for requests.
func NewRequestQueue() *ReqsCollection {
	return &ReqsCollection{
		requests: make(map[ReqID]*ReqInfo),
		queue:    newPrioQueue(),
		mutex:    new(sync.RWMutex),
	}
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

	reqs.queue.pushRequest(req)
	reqs.mutex.Lock()
	reqs.requests[req.ID] = req
	reqs.mutex.Unlock()

	return req.ID, nil
}

// CloseRequest is part of implementation of Requests interface. It checks that
// request is in WAIT state and changes it to CANCEL or in INPROGRESS state and
// changes it to DONE. NotFoundError may be returned if request with given reqID
// doesn't exist in the queue or ErrModificationForbidden if request is in state
// which can't be closed.
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
		req.State = DONE
		// TODO(mwereski): release worker
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
	reqs.mutex.Lock()
	defer reqs.mutex.Unlock()

	dst, ok := reqs.requests[src.ID]
	if !ok {
		return NotFoundError("Request")
	}
	if !modificationPossible(dst.State) {
		return ErrModificationForbidden
	}
	if src.Priority == dst.Priority &&
		src.ValidAfter.Equal(dst.ValidAfter) &&
		src.Deadline.Equal(dst.Deadline) {
		return nil
	}
	// TODO(mwereski): Check if user has rights to set given priority.
	if src.Priority != Priority(0) && (src.Priority < HiPrio ||
		src.Priority > LoPrio) {
		return ErrPriority
	}
	deadline := dst.Deadline
	if !src.Deadline.IsZero() {
		if src.Deadline.Before(time.Now().UTC()) {
			return ErrDeadlineInThePast
		}
		deadline = src.Deadline
	}
	if (!src.ValidAfter.IsZero()) && !deadline.IsZero() &&
		src.ValidAfter.After(deadline) {
		return ErrInvalidTimeRange
	}

	if src.Priority != Priority(0) {
		reqs.queue.setRequestPriority(dst, src.Priority)
		dst.Priority = src.Priority
	}
	if !src.ValidAfter.IsZero() {
		dst.ValidAfter = src.ValidAfter
	}
	dst.Deadline = deadline
	// TODO(mwereski): check if request is ready to go.
	return nil
}

// GetRequestInfo is part of implementation of Requests interface. It returns
// ReqInfo for given request ID or NotFoundError if request with given ID doesn't
// exits in the collection.
func (reqs *ReqsCollection) GetRequestInfo(reqID ReqID) (ReqInfo, error) {
	reqs.mutex.RLock()
	defer reqs.mutex.RUnlock()
	req, ok := reqs.requests[reqID]
	if !ok {
		return ReqInfo{}, NotFoundError("Request")
	}
	return *req, nil
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
