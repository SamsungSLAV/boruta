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
	"crypto/rsa"
	"fmt"

	. "git.tizen.org/tools/boruta"
	"git.tizen.org/tools/muxpi/sw/nanopi/stm"
)

// Rusalka implements Dryad interface. It is intended to be used on NanoPi connected to MuxPi.
// It is not safe for concurrent use.
type Rusalka struct {
	Dryad
	dryadUser         *borutaUser
	cancelMaintenance context.CancelFunc
}

// NewRusalka returns Dryad interface to Rusalka.
func NewRusalka(username string, groups []string) Dryad {
	return &Rusalka{
		dryadUser: newBorutaUser(username, groups),
	}
}

// PutInMaintenance is part of implementation of Dryad interface.
// Connection to STM is being opened only for the maintenance actions.
// Otherwise it may make it unusable for other STM users. It is closed
// when blinkMaintenanceLED exits.
func (r *Rusalka) PutInMaintenance(msg string) error {
	// Connection to STM is closed in blinkMaintenanceLED().
	err := stm.Open()
	if err != nil {
		return err
	}
	err = printMessage(msg)
	if err != nil {
		return err
	}
	var ctx context.Context
	ctx, r.cancelMaintenance = context.WithCancel(context.Background())
	go blinkMaintenanceLED(ctx)
	return nil
}

// Prepare is part of implementation of Dryad interface. Call to Prepare stops LED blinking.
func (r *Rusalka) Prepare() (key *rsa.PrivateKey, err error) {
	// Stop maintenance.
	if r.cancelMaintenance != nil {
		r.cancelMaintenance()
		r.cancelMaintenance = nil
	}
	// Remove/Add user.
	err = r.dryadUser.delete()
	if err != nil {
		return nil, fmt.Errorf("user removal failed: %s", err)
	}
	err = r.dryadUser.add()
	if err != nil {
		return nil, fmt.Errorf("user creation failed: %s", err)
	}
	// Verify user's existance.
	err = r.dryadUser.update()
	if err != nil {
		return nil, fmt.Errorf("user information update failed: %s", err)
	}
	// Prepare SSH access.
	return r.dryadUser.generateAndInstallKey()
}

// Healthcheck is part of implementation of Dryad interface.
func (r *Rusalka) Healthcheck() (err error) {
	err = stm.Open()
	if err != nil {
		return err
	}
	return stm.Close()
}
