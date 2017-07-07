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

// File requests/errors.go provides errors that may occur when handling requests.

package requests

import "errors"

var (
	// ErrPriority means that requested priority is out of bounds.
	// Either general bounds (HiPrio, LoPrio) or allowed priority bounds for
	// request owner.
	ErrPriority = errors.New("requested priority out of bounds")
	// ErrInvalidTimeRange means that requested ValidAfter date is after Deadline value.
	ErrInvalidTimeRange = errors.New("requested time range illegal - ValidAfter must be before Deadline")
	// ErrDeadlineInThePast means that requested Deadline date is in the past.
	ErrDeadlineInThePast = errors.New("Deadline in the past")
	// ErrWorkerNotAssigned means that user tries to call method which regards
	// worker although worker wasn't assigned yet.
	ErrWorkerNotAssigned = errors.New("worker not assigned")
	// ErrModificationForbidden means that user tries to modify request which
	// is in state that forbids any changes.
	ErrModificationForbidden = errors.New("action cannot be executed in current state")
)
