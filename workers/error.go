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

package workers

import (
	"errors"

	"github.com/SamsungSLAV/boruta"
)

var (
	// ErrNotImplemented is returned when function is not implemented yet.
	ErrNotImplemented = errors.New("function not implemented")
	// ErrMissingUUID is returned when Register is called
	// with caps, which do not contain "UUID" field.
	ErrMissingUUID = errors.New("Capabilities are missing UUID entry")
	// ErrWorkerNotFound is returned when UUID argument does not match any worker on the list.
	ErrWorkerNotFound = boruta.NotFoundError("Worker")
	// ErrInMaintenance is returned when SetFail has been called for Worker in MAINTENANCE state.
	ErrInMaintenance = errors.New("It is forbidden to set FAIL state when Worker is in MAINTENANCE state")
	// ErrNotInFailOrMaintenance is returned when Deregister is called for a worker not in FAIL or MAINTENANCE state.
	// Only workers in FAIL or MAINTENANCE state can be removed from the list.
	ErrNotInFailOrMaintenance = errors.New("Worker is not in FAIL or MAINTENANCE state")
	// ErrWrongStateArgument is returned when SetState is called with incorrect state argument.
	// Worker state can be changed by Admin to IDLE or MAINTENANCE only.
	ErrWrongStateArgument = errors.New("Only state changes to IDLE and MAINTENANCE are allowed")
	// ErrForbiddenStateChange is returned when transition from state, Worker is in,
	// to state, SetState has been called with, is forbidden.
	ErrForbiddenStateChange = errors.New("Invalid state transition was requested")
	// ErrNoMatchingWorker is returned when there is no worker matching groups nor
	// capabilities required by request.
	ErrNoMatchingWorker = errors.New("No matching worker")
	// ErrMissingIP is returned when Register is called with either dryad or sshd
	// address missing IP value.
	ErrMissingIP = errors.New("IP address is missing from address")
	// ErrMissingPort is returned when Register is called with either dryad or sshd
	// address missing Port value.
	ErrMissingPort = errors.New("Port is missing from address")
	// ErrWorkerBusy is returned when worker is preparing to enter IDLE or MAINTENANCE state
	// which requires time consuming operations to be run on Dryad. During this preparations
	// Worker is blocked and cannot change state.
	ErrWorkerBusy = errors.New("worker is busy")
)
