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

// File matcher/jobs.go contains implementation JobsManagerImpl of JobsManager
// interface. It creates new jobs, stores them, make their data available and
// allows to finish them.

package matcher

import (
	"sync"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/tunnels"
	"git.tizen.org/tools/boruta/workers"
)

const defaultDryadUsername = "boruta-user"

// JobsManagerImpl manages jobs. It is an implementation of JobsManager interface.
// It provides methods for creation of new jobs, getting their data and finishing
// them.
type JobsManagerImpl struct {
	JobsManager
	// jobs stores all running jobs indexed by ID of the worker they are running on.
	jobs map[boruta.WorkerUUID]*workers.Job
	// workers provides access to workers.
	workers WorkersManager
	// mutex protects JobsManagerImpl from concurrent access.
	mutex *sync.RWMutex
	// hack for mocking purposes
	newTunnel func() tunnels.Tunneler
}

// newTunnel provides default implementation of Tunneler interface.
// It uses tunnels package implementation of Tunnel.
// The function is set as JobsManagerImpl.newTunnel. Field can be replaced
// by another function providing Tunneler for tests purposes.
func newTunnel() tunnels.Tunneler {
	return new(tunnels.Tunnel)
}

// NewJobsManager creates and returns new JobsManagerImpl structure.
func NewJobsManager(w WorkersManager) JobsManager {
	return &JobsManagerImpl{
		jobs:      make(map[boruta.WorkerUUID]*workers.Job),
		workers:   w,
		mutex:     new(sync.RWMutex),
		newTunnel: newTunnel,
	}
}

// Create method creates a new job for the request and the worker. It also prepares
// communication to Dryad by creating a tunnel. It is a part of JobsManager
// interface implementation.
func (m *JobsManagerImpl) Create(req boruta.ReqID, worker boruta.WorkerUUID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	_, present := m.jobs[worker]
	if present {
		return ErrJobAlreadyExists
	}

	addr, err := m.workers.GetWorkerSSHAddr(worker)
	if err != nil {
		return err
	}
	key, err := m.workers.GetWorkerKey(worker)
	if err != nil {
		return err
	}
	t := m.newTunnel()
	err = t.Create(nil, addr)
	if err != nil {
		return err
	}

	job := &workers.Job{
		Access: boruta.AccessInfo{
			Addr: t.Addr(),
			Key:  key,
			// TODO (m.wereski) Acquire username from config.
			Username: defaultDryadUsername,
		},
		Tunnel: t,
		Req:    req,
	}
	m.jobs[worker] = job

	return nil
}

// Get returns job information related to the worker ID or error if no job for
// that worker was found. It is a part of JobsManager interface implementation.
func (m *JobsManagerImpl) Get(worker boruta.WorkerUUID) (*workers.Job, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	job, present := m.jobs[worker]
	if !present {
		return nil, boruta.NotFoundError("Job")
	}
	return job, nil
}

// Finish closes the job execution, breaks the tunnel to Dryad and removes job
// from jobs collection.
// The Dryad should be notified and prepared for next job with key regeneration.
// It is a part of JobsManager interface implementation.
func (m *JobsManagerImpl) Finish(worker boruta.WorkerUUID) error {
	defer m.workers.PrepareWorker(worker, true)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	job, present := m.jobs[worker]
	if !present {
		return boruta.NotFoundError("Job")
	}
	job.Tunnel.Close()
	// TODO log an error in case of tunnel closing failure. Nothing more can be done.
	delete(m.jobs, worker)
	return nil
}
