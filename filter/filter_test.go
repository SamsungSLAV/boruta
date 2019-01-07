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
	var statesTests = [...]struct {
		state string
		name  string
	}{
		{
			state: "WAITING",
			name:  "ALLCAPS",
		},
		{
			state: "waiting",
			name:  "smallcaps",
		},
		{
			state: "Waiting",
			name:  "CamelCase",
		},
		{
			state: string(boruta.WAIT),
			name:  "default",
		},
	}
	priority := boruta.HiPrio.String()
	filter := &Requests{
		State:    string(boruta.WAIT),
		Priority: priority,
	}
	for _, tc := range statesTests {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(filter, NewRequests(tc.state, priority))
		})
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
