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

// File filter/filter_test.go contains tests for requests and workers filters.

package filter

import (
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	assert := assert.New(t)
	state := string(boruta.WAIT)
	priority := boruta.HiPrio.String()
	filter := &Requests{
		State:    state,
		Priority: priority,
	}
	assert.Equal(filter, NewRequests(state, priority))
}

func TestMatch(t *testing.T) {
	assert := assert.New(t)
	req := boruta.ReqInfo{
		ID:       1,
		Priority: (boruta.HiPrio + boruta.LoPrio) / 2,
		State:    boruta.WAIT,
	}

	var statesTests = [...]struct {
		state  string
		result bool
	}{
		{
			state:  string(boruta.WAIT),
			result: true,
		},
		{
			state:  string(boruta.INVALID),
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

	var filter Requests
	for _, stest := range statesTests {
		filter.State = stest.state
		for _, ptest := range priorityTests {
			filter.Priority = ptest.priority
			assert.Equal(stest.result && ptest.result, filter.Match(&req))
		}
	}
	assert.False(filter.Match(nil))
	assert.False(filter.Match(5))
}

func groups(g ...boruta.Group) boruta.Groups {
	return g
}

func caps(v1, v2 string) boruta.Capabilities {
	return boruta.Capabilities{
		"arch":    v1,
		"display": v2,
	}
}

func TestNewWorkers(t *testing.T) {
	assert := assert.New(t)
	g := groups(boruta.Group("foo"), boruta.Group("bar"))
	c := caps("armv7l", "true")
	filter := &Workers{
		Groups:       g,
		Capabilities: c,
	}
	assert.Equal(filter, NewWorkers(g, c))
}
