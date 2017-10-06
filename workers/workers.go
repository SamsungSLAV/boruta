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
	"crypto/rsa"
	"net"

	. "git.tizen.org/tools/boruta"
)

// UUID denotes a key in Capabilities where WorkerUUID is stored.
const UUID string = "UUID"

// mapWorker is used by WorkerList to store all
// (public and private) structures representing Worker.
type mapWorker struct {
	WorkerInfo
	ip  net.IP
	key *rsa.PrivateKey
}

// WorkerList implements Superviser and Workers interfaces.
// It manages a list of Workers.
type WorkerList struct {
	Superviser
	Workers
	workers map[WorkerUUID]*mapWorker
}

// NewWorkerList returns a new WorkerList with all fields set.
func NewWorkerList() *WorkerList {
	return &WorkerList{
		workers: make(map[WorkerUUID]*mapWorker),
	}
}

// Register is an implementation of Register from Superviser interface.
// UUID, which identifies Worker, must be present in caps.
func (wl *WorkerList) Register(caps Capabilities) error {
	capsUUID, present := caps[UUID]
	if !present {
		return ErrMissingUUID
	}
	uuid := WorkerUUID(capsUUID)
	worker, registered := wl.workers[uuid]
	if registered {
		// Subsequent Register calls update the caps.
		worker.Caps = caps
	} else {
		wl.workers[uuid] = &mapWorker{
			WorkerInfo: WorkerInfo{
				WorkerUUID: uuid,
				State:      MAINTENANCE,
				Caps:       caps,
			}}
	}
	return nil
}

// SetFail is an implementation of SetFail from Superviser interface.
//
// TODO(amistewicz): WorkerList should process the reason and store it.
func (wl *WorkerList) SetFail(uuid WorkerUUID, reason string) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	if worker.State == MAINTENANCE {
		return ErrInMaintenance
	}
	worker.State = FAIL
	return nil
}

// SetState is an implementation of SetState from Workers interface.
func (wl *WorkerList) SetState(uuid WorkerUUID, state WorkerState) error {
	// Only state transitions to IDLE or MAINTENANCE are allowed.
	if state != MAINTENANCE && state != IDLE {
		return ErrWrongStateArgument
	}
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	// State transitions to IDLE are allowed from MAINTENANCE state only.
	if state == IDLE && worker.State != MAINTENANCE {
		return ErrForbiddenStateChange
	}
	worker.State = state
	return nil
}

// SetGroups is an implementation of SetGroups from Workers interface.
func (wl *WorkerList) SetGroups(uuid WorkerUUID, groups Groups) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	worker.Groups = groups
	return nil
}

// Deregister is an implementation of Deregister from Workers interface.
func (wl *WorkerList) Deregister(uuid WorkerUUID) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	if worker.State != MAINTENANCE {
		return ErrNotInMaintenance
	}
	delete(wl.workers, uuid)
	return nil
}

// convertToSlice converts given map to slice.
// It is a helper function of ListWorkers.
func convertToSlice(workers map[WorkerUUID]*mapWorker) []WorkerInfo {
	all := make([]WorkerInfo, 0, len(workers))
	for _, worker := range workers {
		all = append(all, worker.WorkerInfo)
	}
	return all
}

// isCapsMatching returns true if all capabilities in src are satisfied by capabilities in dest
// and false in any other case.
//
// TODO Caps matching is a complex problem and it should be changed to satisfy usecases below:
// * matching any of the values and at least one:
//   "SERIAL": "57600,115200" should be satisfied by "SERIAL": "9600, 38400, 57600"
// * match value in range:
//   "VOLTAGE": "2.9-3.6" should satisfy "VOLTAGE": "3.3"
func isCapsMatching(src, dest Capabilities) bool {
	for srcKey, srcValue := range src {
		destValue, ok := dest[srcKey]
		if !ok {
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

// removeFromSlice replaces i-th element with the last one and returns slice shorter by one.
func removeFromSlice(workers []WorkerInfo, i int) []WorkerInfo {
	l := len(workers) - 1 // Index of last element of the slice.
	if i != l {
		workers[i] = workers[l]
	}
	return workers[:l]
}

// filterCaps returns all workers matching given capabilities.
// It is a helper function of ListWorkers.
func filterCaps(workers []WorkerInfo, caps Capabilities) []WorkerInfo {
	if caps == nil || len(caps) == 0 {
		return workers
	}
	// range is not used here as workers reference and parameter i
	// are modified within the loop.
	for i := 0; i < len(workers); i++ {
		worker := &workers[i]
		if !isCapsMatching(caps, worker.Caps) {
			workers = removeFromSlice(workers, i)
			i-- // Ensure that no worker will be skipped.
		}
	}
	return workers
}

// filterGroups returns all workers matching given groups.
// It is a helper function of ListWorkers.
func filterGroups(workers []WorkerInfo, groups Groups) []WorkerInfo {
	if groups == nil || len(groups) == 0 {
		return workers
	}
	groupsMatcher := make(map[Group]interface{})
	for _, group := range groups {
		groupsMatcher[group] = nil
	}
	// range is not used here as workers reference and parameter i
	// are modified within the loop.
	for i := 0; i < len(workers); i++ {
		worker := &workers[i]
		accept := false
		for _, workerGroup := range worker.Groups {
			_, ok := groupsMatcher[workerGroup]
			if ok {
				accept = true
				break
			}
		}
		if !accept {
			workers = removeFromSlice(workers, i)
			i-- // Ensure that no worker will be skipped.
		}
	}
	return workers
}

// ListWorkers is an implementation of ListWorkers from Workers interface.
// It lists all workers when both:
// * any of the groups is matching (or groups is nil)
// * all of the caps is matching (or caps is nil)
func (wl *WorkerList) ListWorkers(groups Groups, caps Capabilities) ([]WorkerInfo, error) {
	return filterGroups(filterCaps(convertToSlice(wl.workers), caps), groups), nil
}

// GetWorkerInfo is an implementation of GetWorkerInfo from Workers interface.
func (wl *WorkerList) GetWorkerInfo(uuid WorkerUUID) (WorkerInfo, error) {
	worker, ok := wl.workers[uuid]
	if !ok {
		return WorkerInfo{}, ErrWorkerNotFound
	}
	return worker.WorkerInfo, nil
}

// SetWorkerIP stores ip in the worker structure referenced by uuid.
// It should be called after Register by function which is aware of
// the source of the connection and therefore its IP address.
func (wl *WorkerList) SetWorkerIP(uuid WorkerUUID, ip net.IP) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	worker.ip = ip
	return nil
}

// GetWorkerIP retrieves IP address from the internal structure.
func (wl *WorkerList) GetWorkerIP(uuid WorkerUUID) (net.IP, error) {
	worker, ok := wl.workers[uuid]
	if !ok {
		return nil, ErrWorkerNotFound
	}
	return worker.ip, nil
}

// SetWorkerKey stores private key in the worker structure referenced by uuid.
// It is safe to modify key after call to this function.
func (wl *WorkerList) SetWorkerKey(uuid WorkerUUID, key *rsa.PrivateKey) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	// Copy key so that it couldn't be changed outside this function.
	worker.key = new(rsa.PrivateKey)
	*worker.key = *key
	return nil
}

// GetWorkerKey retrieves key from the internal structure.
func (wl *WorkerList) GetWorkerKey(uuid WorkerUUID) (rsa.PrivateKey, error) {
	worker, ok := wl.workers[uuid]
	if !ok {
		return rsa.PrivateKey{}, ErrWorkerNotFound
	}
	return *worker.key, nil
}
