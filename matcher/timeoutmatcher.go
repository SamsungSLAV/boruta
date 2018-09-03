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

// File matcher/timeoutmatcher.go provides TimeoutMatcher structure.
// It implements Matcher interface and it should be used for handling timeout
// events caused by expiration of requests' job timeout.

package matcher

import (
	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

// TimeoutMatcher implements Matcher interface for handling running requests
// timeout.
type TimeoutMatcher struct {
	Matcher
	// requests provides internal boruta access to requests.
	requests RequestsManager
}

// NewTimeoutMatcher creates a new TimeoutMatcher structure.
func NewTimeoutMatcher(r RequestsManager) *TimeoutMatcher {
	return &TimeoutMatcher{
		requests: r,
	}
}

// Notify implements Matcher interface. This method reacts to events passed to
// matcher. Close method is called on RequestsManager for each request.
// Some of the cases might be invalid, because the request's state has been changed
// to DONE or FAILED. Verification of closing conditions is done inside Close method.
func (m TimeoutMatcher) Notify(out []boruta.ReqID) {
	logger.Debugf("TimeoutMatcher notified about following requests: %v", out)
	for _, r := range out {
		m.requests.Close(r)
	}
}
