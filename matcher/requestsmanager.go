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

// File matcher/requestsmanager.go defines RequestManager interface with API
// for taking actions triggered by matcher events on requests structures.

package matcher

import (
	"time"

	"github.com/SamsungSLAV/boruta"
)

// RequestsManager interface defines API for internal boruta management of requests.
type RequestsManager interface {
	// InitIteration starts iteration over requests pending in queue.
	// Method returns error if iterations are already started and have not been
	// terminated with TerminateIteration()
	InitIteration() error
	// TerminateIteration finishes iterating over requests formerly started with InitIteration.
	TerminateIteration()
	// Next gets next ID from request queue.
	// Method returns {ID, true} if there is pending request
	// or {ReqID(0), false} if queue's end has been reached.
	Next() (boruta.ReqID, bool)
	// VerifyIfReady checks if the request is ready to be run on worker.
	VerifyIfReady(boruta.ReqID, time.Time) bool
	// Get retrieves full request information or error if no request is found.
	Get(boruta.ReqID) (boruta.ReqInfo, error)
	// Timeout sets request to TIMEOUT state after Deadline time is exceeded.
	Timeout(boruta.ReqID) error
	// Close closes request setting it in DONE state, closing job
	// and releasing worker after run time of the request has been exceeded.
	Close(boruta.ReqID) error
	// Run starts job performing the request on the worker.
	Run(boruta.ReqID, boruta.WorkerUUID) error
}
