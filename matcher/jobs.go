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

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/tunnels"
	"github.com/SamsungSLAV/boruta/workers"
	"github.com/SamsungSLAV/slav/logger"
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
	logger.WithProperty("req-id", req.String()).WithProperty("worker-id", worker).
		Debug("Creating new Job...")

	m.mutex.Lock()
	defer m.mutex.Unlock()

	j, present := m.jobs[worker]
	if present {
		logger.WithProperty("requestID", req.String()).WithProperty("workerID", worker).
			Errorf("Worker is already processing another req: %s.", j.String())
		return ErrJobAlreadyExists
	}

	addr, err := m.workers.GetWorkerSSHAddr(worker)
	if err != nil {
		logger.WithProperty("requestID", req.String()).WithProperty("workerID", worker).
			Error("Cannot get Worker's SSH address.")
		return err
	}
	key, err := m.workers.GetWorkerKey(worker)
	if err != nil {
		logger.WithProperty("requestID", req.String()).WithProperty("workerID", worker).
			Error("Cannot get Worker's key.")
		return err
	}
	t := m.newTunnel()
	err = t.Create(nil, addr)
	if err != nil {
		logger.WithProperty("requestID", req.String()).WithProperty("workerID", worker).
			Error("Cannot create a new tunnel.")
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
	logger.WithProperty("requestID", req.String()).WithProperty("workerID", worker).WithProperty("job", job).
		Info("New Job created.")
	return nil
}

// Get returns job information related to the worker ID or error if no job for
// that worker was found. It is a part of JobsManager interface implementation.
func (m *JobsManagerImpl) Get(worker boruta.WorkerUUID) (*workers.Job, error) {
	logger.WithProperty("workerID", worker).Debug("Get Job for Worker.")
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	job, present := m.jobs[worker]
	if !present {
		logger.WithProperty("workerID", worker).Warning("No Job found for Worker.")
		return nil, boruta.NotFoundError("Job")
	}
	logger.WithProperty("workerID", worker).WithProperty("Job", job).Debug("Worker's Job found.")
	return job, nil
}

// Finish closes the job execution, breaks the tunnel to Dryad and removes job
// from jobs collection.
// The Dryad should be notified and prepared for next job with key regeneration
// if prepare is true.
// If prepare is false, Dryad should not be prepared. This can happen when Dryad
// failed or is brought to MAINTENANCE state.
// It is a part of JobsManager interface implementation.
func (m *JobsManagerImpl) Finish(worker boruta.WorkerUUID, prepare bool) error {
	logger.WithProperty("workerID", worker).WithProperty("prepare", prepare).
		Debug("Finish Worker's Job.")
	if prepare {
		defer m.workers.PrepareWorker(worker, true)
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	job, present := m.jobs[worker]
	if !present {
		logger.WithProperty("workerID", worker).Warning("No Job found for Worker.")
		return boruta.NotFoundError("Job")
	}
	err := job.Tunnel.Close()
	if err != nil {
		logger.WithProperty("workerID", worker).WithError(err).Error("Cannot close Jobs' tunnel.")
	}

	delete(m.jobs, worker)
	logger.WithProperty("workerID", worker).Debug("Job removed.")
	return nil
}
