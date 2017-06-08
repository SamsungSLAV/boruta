/*
 *  Copyright (c) 2017 Samsung Electronics Co., Ltd All Rights Reserved
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

// Package workers is responsible for worker list management.
package workers

import (
	. "git.tizen.org/tools/boruta"
)

// WorkerList implements Superviser and Workers interfaces.
// It manages a list of Workers.
type WorkerList struct {
	Superviser
	Workers
	workers map[WorkerUUID]*WorkerInfo
}

// NewWorkerList returns a new WorkerList with all fields set.
func NewWorkerList() *WorkerList {
	return &WorkerList{
		workers: make(map[WorkerUUID]*WorkerInfo),
	}
}

// Register is an implementation of Register from Superviser interface.
func (wl *WorkerList) Register(caps Capabilities) error {
	return ErrNotImplemented
}

// SetFail is an implementation of SetFail from Superviser interface.
func (wl *WorkerList) SetFail(uuid WorkerUUID, reason string) error {
	return ErrNotImplemented
}

// SetState is an implementation of SetState from Workers interface.
func (wl *WorkerList) SetState(uuid WorkerUUID, state WorkerState) error {
	return ErrNotImplemented
}

// SetGroups is an implementation of SetGroups from Workers interface.
func (wl *WorkerList) SetGroups(uuid WorkerUUID, groups Groups) error {
	return ErrNotImplemented
}

// Deregister is an implementation of Deregister from Workers interface.
func (wl *WorkerList) Deregister(uuid WorkerUUID) error {
	return ErrNotImplemented
}

// ListWorkers is an implementation of ListWorkers from Workers interface.
func (wl *WorkerList) ListWorkers(groups Groups, caps Capabilities) ([]WorkerInfo, error) {
	return nil, ErrNotImplemented
}

// GetWorkerInfo is an implementation of GetWorkerInfo from Workers interface.
func (wl *WorkerList) GetWorkerInfo(uuid WorkerUUID) (WorkerInfo, error) {
	return WorkerInfo{}, ErrNotImplemented
}
