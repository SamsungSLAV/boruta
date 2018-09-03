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
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math"
	"net"
	"reflect"
	"sort"
	"sync"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/filter"
	"github.com/SamsungSLAV/boruta/rpc/dryad"
	"github.com/SamsungSLAV/slav/logger"
	"golang.org/x/crypto/ssh"
)

// UUID denotes a key in Capabilities where WorkerUUID is stored.
const UUID string = "UUID"

// sizeRSA is a length of the RSA key.
// It is a variable for test purposes.
var sizeRSA = 4096

// backgroundOperationsBufferSize defines buffer size of the channel
// used for communication with background goroutine launched for every
// registered worker. The goroutine processes long operations like:
// preparation of Dryad to work or putting it into maintenance state.
// Goroutines related with API calls use channel to initiate background
// operations. Only one operation is run at the same time. The buffer
// on channel allows non-blocking delegation of these operations.
const backgroundOperationsBufferSize int = 32

// pendingOperation describes status of reader's end of the channel used by
// background routine.
// The got flag indicates if there is any operation pending on channel.
// Only if got is set to true, other fields should be analyzed.
// The open flag indicates if channel is still open. It is set to false
// when channel was closed on writer side (during deregistration of worker).
// The state field contains new state that worker was switched to.
type pendingOperation struct {
	state boruta.WorkerState
	open  bool
	got   bool
}

// backgroundContext aggregates data required by functions running in background
// goroutine to identify context of proper worker. The context is build of:
// c - a reader's end of channel;
// uuid - identificator of worker.
type backgroundContext struct {
	c    <-chan boruta.WorkerState
	uuid boruta.WorkerUUID
}

// mapWorker is used by WorkerList to store all
// (public and private) structures representing Worker.
type mapWorker struct {
	boruta.WorkerInfo
	dryad               *net.TCPAddr
	sshd                *net.TCPAddr
	key                 *rsa.PrivateKey
	backgroundOperation chan boruta.WorkerState
}

// WorkerList implements Superviser and Workers interfaces.
// It manages a list of Workers.
// It implements also WorkersManager from matcher package making it usable
// as interface for acquiring workers by Matcher.
// The implemnetation requires changeListener, which is notified after Worker's
// state changes.
// The dryad.ClientManager allows managing Dryads' clients for key generation.
// One can be created using newDryadClient function.
type WorkerList struct {
	boruta.Superviser
	boruta.Workers
	workers        map[boruta.WorkerUUID]*mapWorker
	mutex          *sync.RWMutex
	changeListener WorkerChange
	newDryadClient func() dryad.ClientManager
}

// newDryadClient provides default implementation of dryad.ClientManager interface.
// It uses dryad package implementation of DryadClient.
// The function is set as WorkerList.newDryadClient. Field can be replaced
// by another function providing dryad.ClientManager for tests purposes.
func newDryadClient() dryad.ClientManager {
	return new(dryad.DryadClient)
}

// NewWorkerList returns a new WorkerList with all fields set.
func NewWorkerList() *WorkerList {
	return &WorkerList{
		workers:        make(map[boruta.WorkerUUID]*mapWorker),
		mutex:          new(sync.RWMutex),
		newDryadClient: newDryadClient,
	}
}

// Register is an implementation of Register from Superviser interface.
// UUID, which identifies Worker, must be present in caps. Both dryadAddress and
// sshAddress must resolve and parse to net.TCPAddr. Neither IP address nor port number
// can not be ommited.
func (wl *WorkerList) Register(caps boruta.Capabilities, dryadAddress string,
	sshAddress string) error {

	capsUUID, present := caps[UUID]
	if !present {
		logger.WithProperty("Capabilities", caps).Error("No UUID field in Capabilities.")
		return ErrMissingUUID
	}
	uuid := boruta.WorkerUUID(capsUUID)

	dryad, err := net.ResolveTCPAddr("tcp", dryadAddress)
	if err != nil {
		logger.WithProperty("Capabilities", caps).
			WithError(fmt.Errorf("invalid dryad address: %s", err)).Error("")
		return fmt.Errorf("invalid dryad address: %s", err)
	}

	logger.WithProperty("DryadTCPAddr", dryad).Debug("dryad addr resolved")

	// dryad.IP is empty if dryadAddress provided port number only.
	if dryad.IP == nil {
		logger.WithProperty("Capabilities", caps).WithError(ErrMissingIP).Error("")
		return ErrMissingIP
	}
	if dryad.Port == 0 {
		logger.WithProperty("Capabilities", caps).Error(ErrMissingPort.Error())
		return ErrMissingPort
	}

	sshd, err := net.ResolveTCPAddr("tcp", sshAddress)
	if err != nil {
		logger.WithProperty("Capabilities", caps).
			Error(fmt.Sprintf("invalid sshd address: %s", err))
		return fmt.Errorf("invalid sshd address: %s", err)
	}

	logger.WithProperty("DryadTCPAddr", sshd).Debug("sshd addr resolved")

	// same as with dryad.IP
	if sshd.IP == nil {
		logger.WithProperty("Capabilities", caps).Error(ErrMissingIP.Error())
		return ErrMissingIP
	}
	if sshd.Port == 0 {
		logger.WithProperty("Capabilities", caps).Error("sshd port is 0" + ErrMissingPort.Error())
		return ErrMissingPort
	}

	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, registered := wl.workers[uuid]
	if registered {
		// Subsequent Register calls update the caps and addresses.
		worker.Caps = caps
		worker.dryad = dryad
		worker.sshd = sshd
		logger.WithProperty("Capabilities", caps).WithProperty("WorkerID", uuid).
			Notice("Worker's Capabilities updated.")
	} else {
		c := make(chan boruta.WorkerState, backgroundOperationsBufferSize)
		wl.workers[uuid] = &mapWorker{
			WorkerInfo: boruta.WorkerInfo{
				WorkerUUID: uuid,
				State:      boruta.MAINTENANCE,
				Caps:       caps,
			},
			dryad:               dryad,
			sshd:                sshd,
			backgroundOperation: c,
		}
		go wl.backgroundLoop(backgroundContext{c: c, uuid: uuid})
		logger.WithProperty("Capabilities", caps).WithProperty("WorkerID", uuid).
			Notice("Worker registered.")
	}
	return nil
}

// SetFail is an implementation of SetFail from Superviser interface.
//
// TODO(amistewicz): WorkerList should process the reason and store it.
func (wl *WorkerList) SetFail(uuid boruta.WorkerUUID, reason string) error {
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerId", uuid).WithProperty("reason", reason).
			WithError(ErrWorkerNotFound).Error("Worker not found.")
		return ErrWorkerNotFound
	}
	// Ignore entering FAIL state if administrator started maintenance already.
	if worker.State == boruta.MAINTENANCE || worker.State == boruta.BUSY {
		logger.WithProperty("WorkerId", uuid).WithProperty("reason", reason).
			WithError(ErrInMaintenance).Error("Worker in MAINTENANCE state.")
		return ErrInMaintenance
	}
	return wl.setState(uuid, boruta.FAIL)
}

// SetState is an implementation of SetState from Workers interface. Nil return means that there
// were no formal issues to change the state of the worker. Error may occur while communicating
// with Dryad via RPC. In such case state of the worker will be changed to FAIL. It's
// responsibility of the caller to check if state was changed to requested value.
func (wl *WorkerList) SetState(uuid boruta.WorkerUUID, state boruta.WorkerState) error {
	// Only state transitions to IDLE or MAINTENANCE are allowed.
	if state != boruta.MAINTENANCE && state != boruta.IDLE {
		logger.WithProperty("WorkerId", uuid).WithProperty("state", state).
			WithError(ErrWrongStateArgument).Error("Wrong state.")
		return ErrWrongStateArgument
	}
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerId", uuid).WithProperty("state", state).
			WithError(ErrWorkerNotFound).Error("Worker not found.")
		return ErrWorkerNotFound
	}
	// Do nothing if transition to MAINTENANCE state is already ongoing.
	if state == boruta.MAINTENANCE && worker.State == boruta.BUSY {
		return nil
	}
	// Do nothing if transition to IDLE state is already ongoing.
	if state == boruta.IDLE && worker.State == boruta.PREPARE {
		return nil
	}
	// State transitions to IDLE are allowed from MAINTENANCE state only.
	if state == boruta.IDLE && worker.State != boruta.MAINTENANCE {
		logger.WithProperties(logger.Properties{
			"WorkerID": uuid, "state": state, "worker.State": worker.State}).
			WithError(ErrForbiddenStateChange).Error("Forbidden state change.")
		return ErrForbiddenStateChange
	}
	switch state {
	case boruta.IDLE:
		wl.setState(uuid, boruta.PREPARE)
	case boruta.MAINTENANCE:
		wl.setState(uuid, boruta.BUSY)
	}
	return nil
}

// SetGroups is an implementation of SetGroups from Workers interface.
func (wl *WorkerList) SetGroups(uuid boruta.WorkerUUID, groups boruta.Groups) error {
	logger.WithProperty("WorkerId", uuid).WithProperty("groups", groups).
		Debug("Start.")
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerId", uuid).WithProperty("groups", groups).
			WithError(ErrWorkerNotFound).Error("Worker not found.")
		return ErrWorkerNotFound
	}
	worker.Groups = groups
	sort.Strings(worker.Groups)
	if worker.State == boruta.IDLE && wl.changeListener != nil {
		wl.changeListener.OnWorkerIdle(uuid)
	}
	logger.WithProperty("WorkerId", uuid).WithProperty("groups", groups).
		Debug("Worker groups set.")
	return nil
}

// Deregister is an implementation of Deregister from Workers interface.
func (wl *WorkerList) Deregister(uuid boruta.WorkerUUID) error {
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerId", uuid).
			WithError(ErrWorkerNotFound).Error("Worker not found.")
		return ErrWorkerNotFound
	}
	if worker.State != boruta.MAINTENANCE && worker.State != boruta.FAIL {
		logger.WithProperty("WorkerId", uuid).WithProperty("worker.State", worker.State).
			WithError(ErrNotInFailOrMaintenance).Error("Worker's not in FAIL or MAINTENANCE state.")
		return ErrNotInFailOrMaintenance
	}
	close(worker.backgroundOperation)
	delete(wl.workers, uuid)
	logger.WithProperty("WorkerId", uuid).
		Debug("Worker removed.")
	return nil
}

// min takes two uint64 values and returns smaller one.
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// ListWorkers is an implementation of ListWorkers from Workers interface.
func (wl *WorkerList) ListWorkers(filter boruta.ListFilter, info *boruta.SortInfo,
	paginator *boruta.WorkersPaginator) ([]boruta.WorkerInfo, *boruta.ListInfo, error) {

	if paginator == nil {
		paginator = &boruta.WorkersPaginator{
			Limit: boruta.MaxPageLimit,
		}
	}
	if paginator.Limit == 0 || paginator.Limit > boruta.MaxPageLimit {
		paginator.Limit = boruta.MaxPageLimit
	}

	wl.mutex.RLock()
	var pagWorker *mapWorker
	var found bool
	if pagWorker, found = wl.workers[paginator.ID]; paginator.ID != boruta.WorkerUUID("") && !found {
		wl.mutex.RUnlock()
		return nil, nil, boruta.NotFoundError("worker")
	}
	res, err := wl.listWorkers(filter, info, paginator.ID, false)
	wl.mutex.RUnlock()
	if err != nil {
		return nil, nil, err
	}

	var index, elems uint64
	listInfo := &boruta.ListInfo{TotalItems: uint64(len(res))}
	if listInfo.TotalItems == 0 {
		listInfo.RemainingItems = 0
		return res, listInfo, nil
	}

	for i, w := range res {
		if w.WorkerUUID == paginator.ID {
			index = uint64(i)
			break
		}
	}

	if paginator.Direction == boruta.DirectionForward {
		if paginator.ID != boruta.WorkerUUID("") {
			index++
		}
		elems = min(uint64(paginator.Limit), listInfo.TotalItems-index)
		listInfo.RemainingItems = listInfo.TotalItems - index - elems
	} else {
		if paginator.ID == boruta.WorkerUUID("") {
			index = uint64(len(res))
		}
		elems = min(uint64(paginator.Limit), index)
		listInfo.RemainingItems = index - elems
		index -= elems
	}
	if filter != nil && !reflect.ValueOf(filter).IsNil() && paginator.ID != boruta.WorkerUUID("") &&
		!filter.Match(&pagWorker.WorkerInfo) {
		listInfo.TotalItems--
	}

	return res[index : index+elems], listInfo, nil
}

// listWorkers lists all workers when both:
// * any of the groups is matching (or groups is nil)
// * all of the caps is matching (or caps is nil)
// Caller of this method should own the mutex.
func (wl *WorkerList) listWorkers(filter boruta.ListFilter, info *boruta.SortInfo,
	pid boruta.WorkerUUID, onlyIdle bool) ([]boruta.WorkerInfo, error) {

	sorter, err := newSorter(info)
	if err != nil {
		return nil, err
	}
	matching := make([]boruta.WorkerInfo, 0, len(wl.workers))
	if filter != nil && reflect.ValueOf(filter).IsNil() {
		filter = nil
	}

	for _, worker := range wl.workers {
		if filter == nil || filter.Match(&worker.WorkerInfo) || worker.WorkerUUID == pid {
			if onlyIdle && (worker.State != boruta.IDLE) {
				continue
			}
			matching = append(matching, worker.WorkerInfo)
		}
	}

	sorter.list = matching
	sort.Sort(sorter)
	logger.Debugf("Matching workers: %v", matching)
	return matching, nil
}

// GetWorkerInfo is an implementation of GetWorkerInfo from Workers interface.
func (wl *WorkerList) GetWorkerInfo(uuid boruta.WorkerUUID) (boruta.WorkerInfo, error) {
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", uuid).
			WithError(ErrWorkerNotFound).Warning("Worker not found.")
		return boruta.WorkerInfo{}, ErrWorkerNotFound
	}
	logger.WithProperty("WorkerID", uuid).
		Debugf("Return info: %v.", worker.WorkerInfo)
	return worker.WorkerInfo, nil
}

// getWorkerAddr retrieves IP address from the internal structure.
func (wl *WorkerList) getWorkerAddr(uuid boruta.WorkerUUID) (net.TCPAddr, error) {

	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", uuid).
			WithError(ErrWorkerNotFound).Warning("Worker not found.")
		return net.TCPAddr{}, ErrWorkerNotFound
	}
	logger.WithProperty("WorkerID", uuid).
		Debugf("Return Addr: %v", worker.dryad)
	return *worker.dryad, nil
}

// GetWorkerSSHAddr retrieves address of worker's ssh daemon from the internal structure.
func (wl *WorkerList) GetWorkerSSHAddr(uuid boruta.WorkerUUID) (net.TCPAddr, error) {
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", uuid).
			WithError(ErrWorkerNotFound).Warning("Worker not found.")
		return net.TCPAddr{}, ErrWorkerNotFound
	}
	logger.WithProperty("WorkerID", uuid).
		Debugf("Return SSH Addr: %v", worker.sshd)
	return *worker.sshd, nil
}

// setWorkerKey stores private key in the worker structure referenced by uuid.
// It is safe to modify key after call to this function.
func (wl *WorkerList) setWorkerKey(uuid boruta.WorkerUUID, key *rsa.PrivateKey) error {
	wl.mutex.Lock()
	defer wl.mutex.Unlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", uuid).WithProperty("key", *key).
			WithError(ErrWorkerNotFound).Warning("Worker not found.")
		return ErrWorkerNotFound
	}
	// Copy key so that it couldn't be changed outside this function.
	worker.key = new(rsa.PrivateKey)
	*worker.key = *key
	logger.WithProperty("WorkerID", uuid).WithProperty("key", *key).WithProperty("Worker", worker).
		Debug("Key set.")
	return nil
}

// GetWorkerKey retrieves key from the internal structure.
func (wl *WorkerList) GetWorkerKey(uuid boruta.WorkerUUID) (rsa.PrivateKey, error) {
	wl.mutex.RLock()
	defer wl.mutex.RUnlock()
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", uuid).
			WithError(ErrWorkerNotFound).Warning("Worker not found.")
		return rsa.PrivateKey{}, ErrWorkerNotFound
	}
	logger.WithProperty("WorkerID", uuid).
		Debugf("Return key: %v", *worker.key)
	return *worker.key, nil
}

// TakeBestMatchingWorker verifies which IDLE workers can satisfy Groups and
// Capabilities required by the request. Among all matched workers a best worker
// is choosen (least capable worker still fitting request). If a worker is found
// it is put into RUN state and its UUID is returned. An error is returned if no
// matching IDLE worker is found.
// It is a part of WorkersManager interface implementation by WorkerList.
func (wl *WorkerList) TakeBestMatchingWorker(groups boruta.Groups, caps boruta.Capabilities) (bestWorker boruta.WorkerUUID, err error) {

	wl.mutex.Lock()
	defer wl.mutex.Unlock()

	var bestScore = math.MaxInt32

	matching, err := wl.listWorkers(filter.NewWorkers(groups, caps), nil, boruta.WorkerUUID(""),
		true)
	logger.WithProperty("Groups", groups).WithProperty("Capabilities", caps).
		Debugf("Matching workers are: %v.", matching)
	if err != nil {
		panic("listing workers failed: " + err.Error())
	}
	for _, info := range matching {
		logger.WithProperty("Groups", groups).WithProperty("Capabilities", caps).
			Debugf("Verify worker with info: %v.", info)
		if info.State != boruta.IDLE {
			logger.WithProperty("Groups", groups).WithProperty("Capabilities", caps).
				Debugf("State (%s) not IDLE.", info.State)
			continue
		}
		score := len(info.Caps) + len(info.Groups)
		if score < bestScore {
			bestScore = score
			bestWorker = info.WorkerUUID
		}
		logger.WithProperty("Groups", groups).WithProperty("Capabilities", caps).
			Debugf("Score (%d). Currently best score (%d)", score, bestScore)
	}
	if bestScore == math.MaxInt32 {
		logger.WithProperty("type", "WorkerList").WithProperty("method", "TakeBestMatchingWorker").
			WithProperty("Groups", groups).WithProperty("Capabilities", caps).
			Debug("No matching worker found.")
		err = ErrNoMatchingWorker
		return
	}

	logger.WithProperty("Groups", groups).WithProperty("Capabilities", caps).
		Debugf("Setting RUN state on best worker (%v).", bestWorker)
	err = wl.setState(bestWorker, boruta.RUN)
	return
}

// PrepareWorker brings worker into IDLE state and prepares it to be ready for
// running a job. In some of the situations if a worker has been matched for a job,
// but has not been used, there is no need for regeneration of the key. Caller of
// this method can decide (with 2nd parameter) if key generation is required for
// preparing worker.
//
// As key creation can take some time, the method is asynchronous and the worker's
// state might not be changed when it returns.
// It is a part of WorkersManager interface implementation by WorkerList.
func (wl *WorkerList) PrepareWorker(uuid boruta.WorkerUUID, withKeyGeneration bool) error {
	logger.WithProperty("WorkerID", uuid).WithProperty("withKeyGeneration", withKeyGeneration).
		Debug("Start.")
	wl.mutex.Lock()
	defer wl.mutex.Unlock()

	worker, ok := wl.workers[uuid]
	if !ok {
		return ErrWorkerNotFound
	}
	if worker.State != boruta.RUN {
		return ErrForbiddenStateChange
	}

	if !withKeyGeneration {
		logger.WithProperty("WorkerID", worker).WithProperty("withKeyGeneration", withKeyGeneration).
			Debug("Setting state to IDLE.")
		return wl.setState(uuid, boruta.IDLE)
	}
	return wl.setState(uuid, boruta.PREPARE)
}

// setState changes state of worker. It does not contain any verification if change
// is feasible. It should be used only for internal boruta purposes. It must be
// called inside WorkerList critical section guarded by WorkerList.mutex.
func (wl *WorkerList) setState(uuid boruta.WorkerUUID, state boruta.WorkerState) error {
	worker, ok := wl.workers[uuid]
	if !ok {
		logger.WithProperty("WorkerID", worker).
			WithError(ErrWorkerNotFound).
			Error("Worker not found.")
		return ErrWorkerNotFound
	}
	// Send information about changing state to the background loop to possible break some operations.
	worker.backgroundOperation <- state

	if wl.changeListener != nil {
		if state == boruta.IDLE {
			logger.WithProperty("WorkerID", uuid).
				Debug("Notifying changeListener OnWorkerIdle.")
			wl.changeListener.OnWorkerIdle(uuid)
		}
		// Inform that Job execution was possibly broken when changing RUN state
		// to any other than IDLE or PREPARE.
		if worker.State == boruta.RUN && state != boruta.IDLE && state != boruta.PREPARE {
			logger.WithProperty("WorkerID", uuid).
				Debug("Notifying changeListener OnWorkerFail.")
			wl.changeListener.OnWorkerFail(uuid)
		}
	}
	logger.WithProperty("WorkerID", uuid).
		Debugf("Setting Worker's state to (%s). Former state was (%s).", state, worker.State)
	worker.State = state
	return nil
}

// prepareKeyAndSetState prepares private RSA key for the worker and sets worker
// into IDLE state in case of success. In case of failure of key preparation,
// worker is put into FAIL state instead.
func (wl *WorkerList) prepareKeyAndSetState(bc backgroundContext) (op pendingOperation) {
	var err error
	op, err = wl.prepareKey(bc)
	if op.got {
		return
	}

	wl.mutex.Lock()
	defer wl.mutex.Unlock()

	if op = checkPendingOperation(bc.c); op.got {
		return
	}

	if err != nil {
		// TODO log error.
		wl.setState(bc.uuid, boruta.FAIL)
		return
	}
	wl.setState(bc.uuid, boruta.IDLE)
	return
}

// prepareKey generates key, installs public part on worker and stores private part in WorkerList.
func (wl *WorkerList) prepareKey(bc backgroundContext) (op pendingOperation, err error) {
	if op = checkPendingOperation(bc.c); op.got {
		return
	}
	addr, err := wl.getWorkerAddr(bc.uuid)
	if op = checkPendingOperation(bc.c); op.got || err != nil {
		return
	}
	client := wl.newDryadClient()
	err = client.Create(&addr)
	if err != nil {
		op = checkPendingOperation(bc.c)
		return
	}
	defer client.Close()
	if op = checkPendingOperation(bc.c); op.got {
		return
	}
	key, err := rsa.GenerateKey(rand.Reader, sizeRSA)
	if op = checkPendingOperation(bc.c); op.got || err != nil {
		return
	}
	pubKey, err := ssh.NewPublicKey(&key.PublicKey)
	logger.Debugf("SSH Key: %+v", pubKey)
	if op = checkPendingOperation(bc.c); op.got || err != nil {
		logger.WithError(err).Error("Failed to generate Public SHH  Key.")
		return
	}
	err = client.Prepare(&pubKey)
	if op = checkPendingOperation(bc.c); op.got || err != nil {
		logger.WithProperty("Public Key:", &pubKey).WithError(err).Error("Failed to prepare Dryad.")
		return
	}
	err = wl.setWorkerKey(bc.uuid, key)
	if err != nil {
		logger.WithError(err).Error("Failed to SetWorkerKey.")
	}
	op = checkPendingOperation(bc.c)
	return
}

// putInMaintenanceWorker puts Dryad into maintenance mode and sets worker
// into MAINTENANCE state in case of success. In case of failure of entering
// maintenance mode, worker is put into FAIL state instead.
func (wl *WorkerList) putInMaintenanceWorker(bc backgroundContext) (op pendingOperation) {
	var err error
	op, err = wl.putInMaintenance(bc)
	if op.got {
		return
	}

	wl.mutex.Lock()
	defer wl.mutex.Unlock()

	if op = checkPendingOperation(bc.c); op.got {
		return
	}

	if err != nil {
		logger.WithError(err).
			Error("Error putting Worker in maintenance. Setting Worker's state to FAIL.")
		wl.setState(bc.uuid, boruta.FAIL)
		return
	}
	logger.Error("Setting Worker's state to MAINTENANCE.")
	wl.setState(bc.uuid, boruta.MAINTENANCE)
	return
}

// putInMaintenance orders Dryad to enter maintenance mode.
func (wl *WorkerList) putInMaintenance(bc backgroundContext) (op pendingOperation, err error) {
	if op = checkPendingOperation(bc.c); op.got {
		return
	}
	addr, err := wl.getWorkerAddr(bc.uuid)
	if op = checkPendingOperation(bc.c); op.got || err != nil {
		logger.WithError(err).Error("Failed to GetWorkerAddr.")
		return
	}
	client := wl.newDryadClient()
	err = client.Create(&addr)
	if err != nil {
		op = checkPendingOperation(bc.c)
		logger.WithError(err).Error("Failed to create new RPC client to Dryad.")
		return
	}
	defer client.Close()
	if op = checkPendingOperation(bc.c); op.got {
		return
	}
	err = client.PutInMaintenance("maintenance")
	op = checkPendingOperation(bc.c)
	return
}

// SetChangeListener sets change listener object in WorkerList. Listener should be
// notified in case of changes of workers' states, when worker becomes IDLE
// or must break its job because of fail or maintenance.
// It is a part of WorkersManager interface implementation by WorkerList.
func (wl *WorkerList) SetChangeListener(listener WorkerChange) {
	wl.changeListener = listener
}

// checkPendingOperation verifies status of the communication channel in a non-blocking way.
// It returns pendingOperation structure containing status of the channel.
func checkPendingOperation(c <-chan boruta.WorkerState) (op pendingOperation) {
	for {
		select {
		case op.state, op.open = <-c:
			op.got = true
			if !op.open {
				return
			}
		default:
			return
		}
	}
}

// backgroundLoop is the main procedure of a background goroutine launched for every registered
// worker. It hangs on channel, waiting for new worker state to be processed or for close
// of the channel (when worker was deregistered).
// If new state is received, proper long operation is launched.
// If long operation has been broken by appearance of the new state on channel or channel closure,
// new state is processed immediately.
// Procedure ends, when channel is closed.
func (wl *WorkerList) backgroundLoop(bc backgroundContext) {
	var op pendingOperation

	for {
		if !op.got {
			op.state, op.open = <-bc.c
		}
		if !op.open {
			// Worker has been deregistered. Ending background loop.
			return
		}
		// Clear op.got flag as we consume received state.
		op.got = false
		switch op.state {
		case boruta.PREPARE:
			op = wl.prepareKeyAndSetState(bc)
		case boruta.BUSY:
			op = wl.putInMaintenanceWorker(bc)
		}
	}
}
