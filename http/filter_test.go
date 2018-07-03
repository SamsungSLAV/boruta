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

// File http/filter_test.go contains tests for RequestFilter.

package http

import (
	"testing"

	. "git.tizen.org/tools/boruta"
	"github.com/stretchr/testify/assert"
)

func TestNewRequestFilter(t *testing.T) {
	assert := assert.New(t)
	state := string(WAIT)
	priority := HiPrio.String()
	filter := &RequestFilter{
		State:    state,
		Priority: priority,
	}
	assert.Equal(filter, NewRequestFilter(state, priority))
}

func TestMatch(t *testing.T) {
	assert := assert.New(t)
	req := ReqInfo{
		ID:       1,
		Priority: (HiPrio + LoPrio) / 2,
		State:    WAIT,
	}

	var statesTests = [...]struct {
		state  string
		result bool
	}{
		{
			state:  string(WAIT),
			result: true,
		},
		{
			state:  string(INVALID),
			result: false,
		},
		{
			state:  "",
			result: true,
		},
	}

	var priorityTests = [...]struct {
		priority string
		result   bool
	}{
		{
			priority: req.Priority.String(),
			result:   true,
		},
		{
			priority: (req.Priority + 1).String(),
			result:   false,
		},
		{
			priority: "",
			result:   true,
		},
	}

	var filter RequestFilter
	for _, stest := range statesTests {
		filter.State = stest.state
		for _, ptest := range priorityTests {
			filter.Priority = ptest.priority
			assert.Equal(stest.result && ptest.result, filter.Match(&req))
		}
	}
	assert.False(filter.Match(nil))
}
