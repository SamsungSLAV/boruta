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

package workers

import (
	"errors"
)

var (
	// ErrNotImplemented is returned when function is not implemented yet.
	ErrNotImplemented = errors.New("function not implemented")
	// ErrMissingUUID is returned when Register is called
	// with caps, which do not contain "UUID" field.
	ErrMissingUUID = errors.New("Capabilities are missing UUID entry")
	// ErrWorkerNotFound is returned when UUID argument does not match any worker on the list.
	ErrWorkerNotFound = errors.New("Worker is not present on the list")
	// ErrInMaintenance is returned when SetFail has been called for Worker in MAINTENANCE state.
	ErrInMaintenance = errors.New("It is forbidden to set FAIL state when Worker is in MAINTENANCE state")
	// ErrNotInMaintenance is returned when Deregister is called for a worker not in MAINTENANCE state.
	// Only workers in MAINTENANCE state can be removed from the list.
	ErrNotInMaintenance = errors.New("Worker is not in MAINTENANCE state")
	// ErrWrongStateArgument is returned when SetState is called with incorrect state argument.
	// Worker state can be changed by Admin to IDLE or MAINTENANCE only.
	ErrWrongStateArgument = errors.New("Only state changes to IDLE and MAINTENANCE are allowed")
	// ErrForbiddenStateChange is returned when transition from state, Worker is in,
	// to state, SetState has been called with, is forbidden.
	ErrForbiddenStateChange = errors.New("Invalid state transition was requested")
)
