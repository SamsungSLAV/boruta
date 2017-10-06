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

// File matcher/workersmanager.go defines WorkersManager interface with API
// for taking actions triggered by matcher events on workers structures.

package matcher

import (
	"crypto/rsa"
	"net"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"
)

// WorkersManager defines API for internal boruta management of workers.
type WorkersManager interface {
	// TakeBestMatchingWorker returns best matching worker that meets a criteria.
	// An error is returned if no matching worker is found.
	TakeBestMatchingWorker(Groups, Capabilities) (WorkerUUID, error)

	// PrepareWorker makes it ready for running a job.
	// Caller of this method can decide (with 2nd parameter) if key generation
	// is required for preparing worker.
	PrepareWorker(worker WorkerUUID, withKeyGeneration bool) error

	// GetWorkerIP returns IP of the worker that can be used for setting up tunnel
	// to the worker. If there is no worker with given WorkerUUID an error
	// is returned.
	GetWorkerIP(WorkerUUID) (net.IP, error)

	// GetWorkerKey returns private RSA key of the worker that can be used for
	// accessing the worker. If there is no worker with given WorkerUUID an error
	// is returned.
	GetWorkerKey(WorkerUUID) (rsa.PrivateKey, error)

	// SetChangeListener stores reference to object, which should be notified
	// in case of changes of workers' states.
	SetChangeListener(workers.WorkerChange)
}
