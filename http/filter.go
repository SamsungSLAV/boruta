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

// File http/filter.go provides implementation of ListFilter interface.

package http

import (
	"strings"

	"github.com/SamsungSLAV/boruta"
)

// WorkersFilter contains Groups and Capabilities to be used to filter workers.
type WorkersFilter struct {
	boruta.Groups
	boruta.Capabilities
}

// RequestFilter implements ListFilter interface. Currently it is possible to
// filter by state and priority.
type RequestFilter struct {
	State    string
	Priority string
}

// NewRequestFilter returns pointer to initialized RequestFilter structure.
func NewRequestFilter(state, priority string) *RequestFilter {
	return &RequestFilter{
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
func (filter *RequestFilter) Match(req *boruta.ReqInfo) bool {
	if req == nil {
		return false
	}

	if filter.State != "" && string(req.State) != filter.State {
		return false
	}

	priority := req.Priority.String()
	if filter.Priority != "" && priority != filter.Priority {
		return false
	}

	return true
}
