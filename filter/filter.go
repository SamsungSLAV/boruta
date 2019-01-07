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

// File filter/filter.go provides implementation of ListFilter interface.

// Package filter provides filters used in listing functions.
package filter

import (
	"strings"

	"github.com/SamsungSLAV/boruta"
)

// Workers contains Groups and Capabilities to be used to filter workers.
type Workers struct {
	boruta.Groups
	boruta.Capabilities
	matcher map[boruta.Group]bool
}

// Requests implements ListFilter interface. Currently it is possible to
// filter by state and priority.
type Requests struct {
	State    string
	Priority string
}

// NewRequests returns pointer to initialized Requests structure.
func NewRequests(state, priority string) *Requests {
	return &Requests{
		State:    strings.TrimSpace(strings.ToUpper(state)),
		Priority: strings.TrimSpace(priority),
	}
}

// Match is implementation of ListFilter interface. It checks if given ReqInfo
// matches ListFilter. For now only exact matches are possible, but in the future
// following functionality should be added:
// * comparison,
// * ranges,
// * one of given,
// * except of.
func (filter *Requests) Match(elem interface{}) bool {
	req, ok := elem.(*boruta.ReqInfo)
	if !ok || req == nil {
		return false
	}

	state := strings.ToUpper(filter.State)
	if state != "" && string(req.State) != state {
		return false
	}

	priority := req.Priority.String()
	if filter.Priority != "" && priority != filter.Priority {
		return false
	}

	return true
}

// makeMatcher is helper function of NewWorkers. It prepares groups matcher.
func makeMatcher(groups boruta.Groups) (ret map[boruta.Group]bool) {
	ret = make(map[boruta.Group]bool)
	for _, group := range groups {
		ret[group] = true
	}
	return
}

// NewWorkers returns pointer to initialized Workers structure.
func NewWorkers(groups boruta.Groups, caps boruta.Capabilities) *Workers {
	return &Workers{
		Groups:       groups,
		Capabilities: caps,
		matcher:      makeMatcher(groups),
	}
}

// isCapsMatching returns true if a worker has Capabilities satisfying caps.
// The worker satisfies caps if and only if one of the following statements is true:
//
// * set of required capabilities is empty,
//
// * every key present in set of required capabilities is present in set of worker's capabilities,
//
// * value of every required capability matches the value of the capability in worker.
//
// TODO Caps matching is a complex problem and it should be changed to satisfy usecases below:
// * matching any of the values and at least one:
//   "SERIAL": "57600,115200" should be satisfied by "SERIAL": "9600, 38400, 57600" (as "57600"
//   matches)
// * match value in range:
//   "VOLTAGE": "2.9-3.6" should satisfy "VOLTAGE": "3.3"
func isCapsMatching(worker *boruta.WorkerInfo, caps boruta.Capabilities) bool {
	if len(caps) == 0 {
		return true
	}
	for srcKey, srcValue := range caps {
		destValue, found := worker.Caps[srcKey]
		if !found {
			// Key is not present in the worker's caps
			return false
		}
		if srcValue != destValue {
			// Capability values do not match
			return false
		}
	}
	return true
}

// isGroupsMatching returns true if a worker belongs to any of groups in matcher.
// Empty matcher is satisfied by every Worker.
func isGroupsMatching(worker *boruta.WorkerInfo, matcher map[boruta.Group]bool) bool {
	if len(matcher) == 0 {
		return true
	}
	for _, workerGroup := range worker.Groups {
		if matcher[workerGroup] {
			return true
		}
	}
	return false
}

// Match is implementation of ListFilter interface. It checks if given WorkerInfo
// matches ListFilter. For now only exact matches are possible, but in the future
// following functionality should be added:
// * comparison,
// * ranges,
// * one of given,
// * except of.
func (filter *Workers) Match(elem interface{}) bool {
	worker, ok := elem.(*boruta.WorkerInfo)

	if !ok || worker == nil {
		return false
	}

	if !isGroupsMatching(worker, filter.matcher) {
		return false
	}

	if !isCapsMatching(worker, filter.Capabilities) {
		return false
	}

	return true
}
