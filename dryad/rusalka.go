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

// Package dryad provides:
// * implementation of Dryad interface
// * utilities to manage Dryad and its users
package dryad

import (
	"context"
	"fmt"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/muxpi/sw/nanopi/stm"
	"github.com/SamsungSLAV/slav/logger"
	"golang.org/x/crypto/ssh"
)

// Rusalka implements Dryad interface. It is intended to be used on NanoPi connected to MuxPi.
// It is not safe for concurrent use.
type Rusalka struct {
	boruta.Dryad
	dryadUser         *borutaUser
	stm               *stmHelper
	cancelMaintenance context.CancelFunc
}

// NewRusalka returns Dryad interface to Rusalka.
func NewRusalka(stmConn stm.Interface, username string, groups []string) boruta.Dryad {
	return &Rusalka{
		dryadUser: newBorutaUser(username, groups),
		stm:       &stmHelper{stmConn},
	}
}

// PutInMaintenance is part of implementation of Dryad interface.
// Connection to STM is being opened only for the maintenance actions.
// Otherwise it may make it unusable for other STM users. It is closed
// when blinkMaintenanceLED exits.
func (r *Rusalka) PutInMaintenance(msg string) error {

	err := r.stm.printMessage(msg)
	if err != nil {
		logger.WithProperty("Message", msg).WithError(err).
			Error("Failed to print on stm display.")
		return err
	}
	var ctx context.Context
	ctx, r.cancelMaintenance = context.WithCancel(context.Background())
	go r.stm.blinkMaintenanceLED(ctx)
	return nil
}

// Prepare is part of implementation of Dryad interface. Call to Prepare stops LED blinking.
func (r *Rusalka) Prepare(key *ssh.PublicKey) (err error) {

	// Stop maintenance.
	if r.cancelMaintenance != nil {
		r.cancelMaintenance()
		r.cancelMaintenance = nil
	}
	// Remove/Add user.
	err = r.dryadUser.delete()
	if err != nil {
		logger.WithProperty("SSH public key", key).WithError(err).
			Error("User removal failed")
		return fmt.Errorf("user removal failed: %s", err)
	}
	err = r.dryadUser.add()
	if err != nil {
		logger.WithProperty("SSH public key", key).WithError(err).
			Error("User creation failed")
		return fmt.Errorf("user creation failed: %s", err)
	}
	// Verify user's existance.
	err = r.dryadUser.update()
	if err != nil {
		logger.WithProperty("SSH public key", key).WithError(err).
			Error("user information update failed")
		return fmt.Errorf("user information update failed: %s", err)
	}
	return r.dryadUser.installKey(key)
}

// Healthcheck is part of implementation of Dryad interface.
func (r *Rusalka) Healthcheck() (err error) {
	return r.stm.powerTick()
}
