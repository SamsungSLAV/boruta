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

// Package workers is responsible for worker list management.
package workers

import (
	"crypto/rsa"
	"net"
	"sync"

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
	mutex   *sync.RWMutex
}

// NewWorkerList returns a new WorkerList with all fields set.
func NewWorkerList() *WorkerList {
	return &WorkerList{
		workers: make(map[WorkerUUID]*mapWorker),
		mutex:   new(sync.RWMutex),
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
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
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
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
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
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
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
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	worker.Groups = groups
	return nil
}

// Deregister is an implementation of Deregister from Workers interface.
func (wl *WorkerList) Deregister(uuid WorkerUUID) error {
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
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
//   "SERIAL": "57600,115200" should be satisfied by "SERIAL": "9600, 38400, 57600"
// * match value in range:
//   "VOLTAGE": "2.9-3.6" should satisfy "VOLTAGE": "3.3"
//
// It is a helper function of ListWorkers.
func isCapsMatching(worker WorkerInfo, caps Capabilities) bool {
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

// isGroupsMatching returns true if a worker belongs to any of groups in groupsMatcher.
// Empty groupsMatcher is satisfied by every Worker.
// It is a helper function of ListWorkers.
func isGroupsMatching(worker WorkerInfo, groupsMatcher map[Group]interface{}) bool {
	if len(groupsMatcher) == 0 {
		return true
	}
	for _, workerGroup := range worker.Groups {
		_, ok := groupsMatcher[workerGroup]
		if ok {
			return true
		}
	}
	return false
}

// ListWorkers is an implementation of ListWorkers from Workers interface.
// It lists all workers when both:
// * any of the groups is matching (or groups is nil)
// * all of the caps is matching (or caps is nil)
func (wl *WorkerList) ListWorkers(groups Groups, caps Capabilities) ([]WorkerInfo, error) {
	matching := make([]WorkerInfo, 0, len(wl.workers))

	groupsMatcher := make(map[Group]interface{})
	for _, group := range groups {
		groupsMatcher[group] = nil
	}

	for _, worker := range wl.workers {
		if isGroupsMatching(worker.WorkerInfo, groupsMatcher) &&
			isCapsMatching(worker.WorkerInfo, caps) {
			matching = append(matching, worker.WorkerInfo)
		}
	}
	return matching, nil
}

// GetWorkerInfo is an implementation of GetWorkerInfo from Workers interface.
func (wl *WorkerList) GetWorkerInfo(uuid WorkerUUID) (WorkerInfo, error) {
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
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
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	worker.ip = ip
	return nil
}

// GetWorkerIP retrieves IP address from the internal structure.
func (wl *WorkerList) GetWorkerIP(uuid WorkerUUID) (net.IP, error) {
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		return nil, ErrWorkerNotFound
	}
	return worker.ip, nil
}

// SetWorkerKey stores private key in the worker structure referenced by uuid.
// It is safe to modify key after call to this function.
func (wl *WorkerList) SetWorkerKey(uuid WorkerUUID, key *rsa.PrivateKey) error {
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
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
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		return rsa.PrivateKey{}, ErrWorkerNotFound
	}
	return *worker.key, nil
}
