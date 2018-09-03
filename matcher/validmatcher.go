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

// File matcher/validmatcher.go provides ValidMatcher structure which implements
// Matcher interface. It should be used for handling events caused by validation
// of requests after ValidAfter time is passed.

package matcher

import (
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

// ValidMatcher implements Matcher interface for handling requests validation.
type ValidMatcher struct {
	Matcher
	// requests provides internal boruta access to requests.
	requests RequestsManager
	// workers provides internal boruta access to workers.
	workers WorkersManager
	// jobs provides internal boruta access to jobs.
	jobs JobsManager
}

// NewValidMatcher creates a new ValidMatcher structure.
func NewValidMatcher(r RequestsManager, w WorkersManager, j JobsManager) *ValidMatcher {
	return &ValidMatcher{
		requests: r,
		workers:  w,
		jobs:     j,
	}
}

// Notify implements Matcher interface. This method reacts on events passed to
// matcher. In this implementation requests' IDs are ignored as requests must be
// matched in order they are placed in requests priority queue.
func (m ValidMatcher) Notify(r []boruta.ReqID) {
	logger.Debugf("ValidMatcher notified about following requests: %v", r)
	// Repeat verification until iterateRequests() returns false indicating that
	// there is no more job to be done.
	for m.iterateRequests() {
	}
}

// iterateRequests visits all requests in order they are placed in requests
// priority queue, verifies if they can be run and tries to match an idle worker.
// Method returns true if iteration should be repeated or false if there is
// nothing more to be done.
func (m ValidMatcher) iterateRequests() bool {
	err := m.requests.InitIteration()
	if err != nil {
		logger.Critical("Critical logic error. No iterations over requests collection should be running.")
		panic("Critical logic error. No iterations over requests collection should be running.")
	}
	defer m.requests.TerminateIteration()

	now := time.Now()
	// Iterate on requests priority queue.
	for rid, rok := m.requests.Next(); rok; rid, rok = m.requests.Next() {
		logger.WithProperty("requestID", rid.String()).Debug("Iterate.")
		// Verify if request is ready to be run.
		if !m.requests.VerifyIfReady(rid, now) {
			logger.WithProperty("requestID", rid.String()).Debug("VerifyIfReady failed.")
			continue
		}
		// Request is ready to be run. Get full information about it.
		req, err := m.requests.Get(rid)
		if err != nil {
			logger.WithProperty("requestID", rid.String()).Debug("Get failed.")
			continue
		}

		// Try finding an idle worker matching requests requirements.
		if m.matchWorkers(req) {
			logger.WithProperty("requestID", rid.String()).Debug("Worker matched.")
			// A match was made. Restarting iterations to process other requests.
			return true
		}
	}
	// All requests have been analyzed. No repetition is required.
	return false
}

// matchWorkers tries to find the best of the idle workers matching capabilities
// and groups of the requests. Best worker is the one with least matching penalty.
// If such worker is found a job is created and the request is processed.
func (m ValidMatcher) matchWorkers(req boruta.ReqInfo) bool {
	logger.WithProperty("requestID", req.ID.String()).Debug("Start.")
	worker, err := m.workers.TakeBestMatchingWorker(req.Owner.Groups, req.Caps)
	if err != nil {
		// No matching worker was found.
		logger.WithProperty("requestID", req.ID.String()).Debug("No matching worker found.")
		return false
	}
	// Match found.
	logger.WithProperty("requestID", req.ID.String()).WithProperty("workerID", worker).
		Info("Matching worker found.")
	err = m.jobs.Create(req.ID, worker)
	if err != nil {
		logger.WithProperty("requestID", req.ID.String()).WithProperty("workerID", worker).
			WithError(err).Error("Job creation failed.")
		goto fail
	}
	err = m.requests.Run(req.ID, worker)
	if err != nil {
		logger.WithProperty("requestID", req.ID.String()).WithProperty("workerID", worker).
			WithError(err).Error("Cannot run request.")
		goto fail
	}
	return true

fail:
	// Creating job failed. Bringing worker back to IDLE state.
	logger.WithProperty("requestID", req.ID.String()).WithProperty("workerID", worker).
		Debug("Prepare worker.")
	m.workers.PrepareWorker(worker, false)
	return false
}
