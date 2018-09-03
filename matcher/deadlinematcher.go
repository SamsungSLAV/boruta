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

// File matcher/deadlinematcher.go provides DeadlineMatcher structure.
// It implements Matcher interface and it should be used for handling timeout
// events caused by expiration of Deadline requests' times.

package matcher

import (
	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

// DeadlineMatcher implements Matcher interface for handling pending requests
// timeout.
type DeadlineMatcher struct {
	Matcher
	// requests provides internal boruta access to requests.
	requests RequestsManager
}

// NewDeadlineMatcher creates a new DeadlineMatcher structure.
func NewDeadlineMatcher(r RequestsManager) *DeadlineMatcher {
	return &DeadlineMatcher{
		requests: r,
	}
}

// Notify implements Matcher interface. This method reacts on events passed to
// matcher. Timeout method is called on RequestsManager for each request.
// Some of the timeouts might be invalid, because the request's state has been
// changed to CANCEL or INPROGRESS; or the Deadline time itself has been changed.
// Verification if timeout conditions are met is done in Timeout() method.
// If changing state to TIMEOUT is not possible Timeout returns an error.
// Any errors are ignored as they are false negatives cases from DeadlineMatcher
// point of view.
func (m DeadlineMatcher) Notify(dead []boruta.ReqID) {
	logger.WithProperty("type", "DeadlineMatcher").WithProperty("method", "Notify").
		Debugf("DeadlineMatcher notified about following requests: %v", dead)
	for _, r := range dead {
		m.requests.Timeout(r)
	}
}
