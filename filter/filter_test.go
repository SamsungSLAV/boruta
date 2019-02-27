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

	var newRequestTests = [...]struct {
		name           string
		ids            []boruta.ReqID
		priorities     []boruta.Priority
		states         []boruta.ReqState
		expectedStates []boruta.ReqState
	}{
		{
			name:       "mix of values",
			ids:        []boruta.ReqID{3, 5, 7, 11},
			priorities: []boruta.Priority{boruta.HiPrio, boruta.Priority(127)},
			states: []boruta.ReqState{
				boruta.WAIT,
				boruta.ReqState("  done  "),
				boruta.ReqState("Failed  "),
				boruta.ReqState("  InVaLiD"),
			},
			expectedStates: []boruta.ReqState{
				boruta.WAIT,
				boruta.DONE,
				boruta.FAILED,
				boruta.INVALID,
			},
		},
		{
			name:           "nil values",
			ids:            nil,
			priorities:     nil,
			states:         nil,
			expectedStates: nil,
		},
		{
			name:           "single values",
			ids:            []boruta.ReqID{42},
			priorities:     []boruta.Priority{boruta.HiPrio},
			states:         []boruta.ReqState{boruta.WAIT},
			expectedStates: []boruta.ReqState{boruta.WAIT},
		},
		{
			name:           "empty slices",
			ids:            []boruta.ReqID{},
			priorities:     []boruta.Priority{},
			states:         []boruta.ReqState{},
			expectedStates: []boruta.ReqState{},
		},
	}

	for _, tcase := range newRequestTests {
		filter := NewRequests(tcase.ids, tcase.priorities, tcase.states)
		assert.NotNil(filter, tcase.name)

		// Verify IDs.
		assert.Len(filter.IDs, len(tcase.ids), tcase.name)
		if len(tcase.ids) > 0 {
			assert.Equal(tcase.ids, filter.IDs, tcase.name)
		} else {
			assert.Nil(filter.IDs)
		}

		// Verify Priorities.
		assert.Len(filter.Priorities, len(tcase.priorities), tcase.name)
		if len(tcase.priorities) > 0 {
			assert.Equal(tcase.priorities, filter.Priorities, tcase.name)
		} else {
			assert.Nil(filter.Priorities)
		}

		// Verify States.
		assert.Len(filter.States, len(tcase.states), tcase.name)
		if len(tcase.states) > 0 {
			assert.Equal(tcase.expectedStates, filter.States, tcase.name)
		} else {
			assert.Nil(filter.States)
		}
	}
}

func TestRequestMatch(t *testing.T) {
	assert := assert.New(t)
	req := boruta.ReqInfo{
		ID:       1,
		Priority: (boruta.HiPrio + boruta.LoPrio) / 2,
		State:    boruta.WAIT,
	}

	var statesTests = [...]struct {
		states []boruta.ReqState
		result bool
	}{
		{
			states: []boruta.ReqState{boruta.WAIT},
			result: true,
		},
		{
			states: []boruta.ReqState{boruta.WAIT, boruta.FAILED},
			result: true,
		},
		{
			states: []boruta.ReqState{boruta.FAILED},
			result: false,
		},
		{
			states: []boruta.ReqState{boruta.FAILED, boruta.INVALID},
			result: false,
		},
		{
			states: nil,
			result: true,
		},
		{
			states: []boruta.ReqState{},
			result: true,
		},
	}

	var priorityTests = [...]struct {
		priorities []boruta.Priority
		result     bool
	}{
		{
			priorities: []boruta.Priority{req.Priority},
			result:     true,
		},
		{
			priorities: []boruta.Priority{req.Priority, req.Priority + 1},
			result:     true,
		},
		{
			priorities: []boruta.Priority{req.Priority + 1},
			result:     false,
		},
		{
			priorities: []boruta.Priority{req.Priority + 1, req.Priority - 1},
			result:     false,
		},
		{
			priorities: []boruta.Priority{},
			result:     true,
		},
		{
			priorities: nil,
			result:     true,
		},
	}

	var idsTests = [...]struct {
		ids    []boruta.ReqID
		result bool
	}{
		{
			ids:    []boruta.ReqID{req.ID},
			result: true,
		},
		{
			ids:    []boruta.ReqID{req.ID, req.ID + 1},
			result: true,
		},
		{
			ids:    []boruta.ReqID{req.ID + 1},
			result: false,
		},
		{
			ids:    []boruta.ReqID{req.ID + 1, req.ID + 2},
			result: false,
		},
		{
			ids:    []boruta.ReqID{},
			result: true,
		},
		{
			ids:    nil,
			result: true,
		},
	}

	var filter Requests
	for _, stest := range statesTests {
		filter.States = stest.states
		for _, ptest := range priorityTests {
			filter.Priorities = ptest.priorities
			for _, idstest := range idsTests {
				filter.IDs = idstest.ids
				assert.Equal(stest.result && ptest.result && idstest.result, filter.Match(&req))
			}
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
	m := make(map[boruta.Group]bool)
	for _, group := range g {
		m[group] = true
	}
	filter := &Workers{
		Groups:       g,
		Capabilities: c,
		matcher:      m,
	}
	assert.Equal(filter, NewWorkers(g, c))
}

func TestWorkerMatch(t *testing.T) {
	assert := assert.New(t)

	newWorker := func(g boruta.Groups, c boruta.Capabilities) *boruta.WorkerInfo {
		return &boruta.WorkerInfo{
			Groups: g,
			Caps:   c,
		}
	}

	empty := boruta.Group("empty")
	all := boruta.Group("all")
	some := boruta.Group("some")
	other := boruta.Group("other")

	var tests = [...]struct {
		worker *boruta.WorkerInfo
		filter *Workers
		result bool
	}{
		{
			worker: newWorker(groups(all), caps("armv7", "true")),
			filter: NewWorkers(nil, nil),
			result: true,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(groups(empty), nil),
			result: false,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(nil, caps("aarch64", "true")),
			result: true,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(nil, make(boruta.Capabilities)),
			result: true,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(make(boruta.Groups, 0), nil),
			result: true,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(groups(all, other), caps("aarch64", "true")),
			result: true,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(groups(other), caps("aarch64", "true")),
			result: false,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(groups(all, other), caps("aarch64", "false")),
			result: false,
		},
		{
			worker: newWorker(groups(all, some), caps("aarch64", "true")),
			filter: NewWorkers(groups(all, other),
				boruta.Capabilities{"foo": "bar"}),
			result: false,
		},
	}

	for _, tcase := range tests {
		assert.Equal(tcase.result, tcase.filter.Match(tcase.worker))
	}

	filter := new(Workers)
	assert.False(filter.Match(nil))
	assert.False(filter.Match(5))
}
