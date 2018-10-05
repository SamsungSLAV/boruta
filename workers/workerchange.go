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

// File workers/workerchange.go defines WorkerChange interface with API
// for notification about changes in workers' states.

package workers

import (
	"github.com/SamsungSLAV/boruta"
)

// WorkerChange defines API for implementation to be informed about
// changes in workers.
type WorkerChange interface {
	// OnWorkerIdle notifies about available idle worker.
	OnWorkerIdle(boruta.WorkerUUID)
	// OnWorkerFail notifies about breaking execution of job by a running worker and
	// putting it into FAIL or MAINTENANCE state.
	OnWorkerFail(boruta.WorkerUUID)
}
