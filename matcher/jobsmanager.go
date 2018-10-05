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

// File matcher/jobsmanager.go defines JobsManager interface with API
// for managing jobs.

package matcher

import (
	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/workers"
)

// JobsManager defines API for internal boruta management of jobs.
type JobsManager interface {
	// Create prepares a new job for the worker.
	Create(boruta.ReqID, boruta.WorkerUUID) error
	// Get returns pointer to a Job from JobsManager or error if no job for
	// the worker is found.
	Get(boruta.WorkerUUID) (*workers.Job, error)
	// Finish cleans up after job is done. The second parameter defines
	// if worker should be prepared for next job.
	Finish(boruta.WorkerUUID, bool) error
}
