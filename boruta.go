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

// Package boruta contains definitions of all interfaces and structs used
// between the main modules of the Boruta system.
// Server - service managing Users, Workers, Requests and Jobs.
// User - a system or a person creating the Requests.
// Worker - MuxPi with a target device, which executes the Jobs from the Server.
package boruta

import (
	"crypto/rsa"
	"time"
)

// ReqState denotes state of the Request.
type ReqState string

const (
	// WAIT - Request is in the Queue waiting for processing.
	WAIT ReqState = "WAITING"
	// INPROGRESS - Request has Job with Worker assigned.
	INPROGRESS ReqState = "IN PROGRESS"
	// CANCEL - Request has been cancelled by the User.
	CANCEL ReqState = "CANCELLED"
	// TIMEOUT - Deadline is past due.
	TIMEOUT ReqState = "TIMEOUT"
	// INVALID - Request can no longer be satisfied.
	INVALID ReqState = "INVALID"
	// DONE - Request has finished execution.
	DONE ReqState = "DONE"
	// FAILED - Worker has failed or has been put into MAINTENANCE state by the Admin.
	FAILED ReqState = "FAILED"
)

// WorkerState denotes state of the Worker.
type WorkerState string

const (
	// MAINTENANCE - Worker will not be assigned any Jobs.
	MAINTENANCE WorkerState = "MAINTENANCE"
	// IDLE - Worker is waiting for the Job.
	IDLE WorkerState = "IDLE"
	// RUN - Job is currently being executed on the Worker.
	RUN WorkerState = "RUNNING"
	// FAIL - An error occured, reported by the Worker itself or the Server.
	FAIL WorkerState = "FAILED"
)

// Capabilities describe the features provided by the Worker and required by the Request.
// They are also known as caps.
type Capabilities map[string]string

// ReqID refers to the Request created by the User.
type ReqID uint64

// Priority is the importance of the Request. Lower - more important.
type Priority uint8

const (
	// HiPrio - most important priority (minimum value).
	HiPrio Priority = 0
	// LoPrio - least important priority (maximum value).
	LoPrio Priority = 15
)

// UserInfo is a definition of the User or the Admin.
type UserInfo struct{}

// ReqInfo describes the Request.
type ReqInfo struct {
	ID       ReqID
	Priority Priority
	// Owner is the User who created the Request.
	Owner UserInfo
	// Deadline is a time after which a Request's State will be set to TIMEOUT
	// if it had not been fulfilled.
	Deadline time.Time
	// ValidAfter is a time before which a Request will not be executed.
	ValidAfter time.Time
	State      ReqState
	Job        *JobInfo
	// Caps are the Worker capabilities required by the Request.
	Caps Capabilities
}

// WorkerUUID refers the Worker on which a Job will execute.
type WorkerUUID string

// JobInfo describes the Job.
type JobInfo struct {
	WorkerUUID WorkerUUID
	// Timeout after which this Job will be terminated.
	Timeout time.Time
}

// Group is a set of Workers.
type Group string

// Groups is a superset of all instances of Group.
type Groups []Group

// AccessInfo contains necessary information to access the Worker.
type AccessInfo struct{}

// WorkerInfo describes the Worker.
type WorkerInfo struct {
	WorkerUUID WorkerUUID
	State      WorkerState
	Groups     Groups
	Caps       Capabilities
}

// ListFilter is used to filter Requests in the Queue.
type ListFilter interface {
	// Match tells if request matches the filter.
	Match(req *ReqInfo) bool
}

// Requests defines an interaction of a user with Requests Queue.
type Requests interface {
	// NewRequest creates a Request with given features and adds it to the Queue.
	// It returns ID of the created Request.
	NewRequest(caps Capabilities, priority Priority, owner UserInfo,
		validAfter time.Time, deadline time.Time) (ReqID, error)
	// CloseRequest sets the Request's State to CANCEL (removes from the Queue)
	// or DONE (finishes the Job).
	CloseRequest(reqID ReqID) error
	// SetRequestPriority sets the Request's Priority after it has been created.
	SetRequestPriority(reqID ReqID, priority Priority) error
	// SetRequestValidAfter sets the Request's ValidAfter after it has been created.
	SetRequestValidAfter(reqID ReqID, validAfter time.Time) error
	// SetRequestDeadline sets the Request's Deadline after it has been created.
	SetRequestDeadline(reqID ReqID, deadline time.Time) error
	// GetRequestInfo returns ReqInfo associated with ReqID.
	GetRequestInfo(reqID ReqID) (ReqInfo, error)
	// ListRequests returns ReqInfo matching the filter
	// or all Requests if empty filter is given.
	ListRequests(filter ListFilter) ([]ReqInfo, error)
	// AcquireWorker returns information necessary to access the Worker reserved by the Request
	// and prolongs access to it. If the Request is in the WAIT state, call to this function
	// will block until the state changes.
	// Users should use ProlongAccess() in order to avoid side-effects.
	AcquireWorker(reqID ReqID) (AccessInfo, error)
	// ProlongAccess sets the Job's Deadline to a predefined time.Duration from the time.Now().
	// It can be called multiple times, but is limited.
	ProlongAccess(reqID ReqID) error
}

// Superviser defines registration and repost actions that can be done by a worker only.
type Superviser interface {
	// Register adds a new Worker to the system in the MAINTENANCE state.
	// Capabilities are set on the Worker and can be changed by subsequent Register calls.
	Register(caps Capabilities) error
	// SetFail notifies the Server about the Failure of the Worker.
	// It can additionally contain non-empty reason of the failure.
	SetFail(uuid WorkerUUID, reason string) error
}

// Workers defines all actions that can be done by users and admins on workers.
// Users (and admins) can also call methods from Requests interface.
type Workers interface {
	// ListWorkers returns a list of all Workers matching Groups and Capabilities
	// or all registered Workers if both arguments are empty.
	ListWorkers(groups Groups, caps Capabilities) ([]WorkerInfo, error)
	// GetWorkerInfo returns WorkerInfo of specified worker.
	GetWorkerInfo(uuid WorkerUUID) (WorkerInfo, error)

	// Following methods are for administrators only.

	// SetState sets the Worker's state to either MAINTENANCE or IDLE.
	SetState(uuid WorkerUUID, state WorkerState) error
	// SetGroups updates the groups parameter of the Worker.
	SetGroups(uuid WorkerUUID, groups Groups) error
	// Deregister removes the Worker from the system.
	// It can only succeed if the Worker is in the MAINTENANCE mode.
	Deregister(uuid WorkerUUID) error
}

// Dryad is a MuxPi management interface used by Boruta.
type Dryad interface {
	// PutInMaintenance prepares MuxPi for administrative action.
	// It blinks LEDs, prints msg on the OLED display, etc.
	PutInMaintenance(msg string) error
	// Prepare creates appropriate user, generates RSA key, installs public key
	// so that it can be used for SSH authentication and returns private key.
	// It removes current instance of the user, etc.
	Prepare() (*rsa.PrivateKey, error)
	// Healthcheck tests Dryad for system state, STM functions and state on MuxPi.
	// It may cause Dryad to call SetFail of Worker interface if the problem detected is critical.
	Healthcheck() error
}
