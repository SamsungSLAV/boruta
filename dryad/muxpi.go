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

package dryad

import (
	"context"
	"time"

	"git.tizen.org/tools/muxpi/sw/nanopi/stm"
)

type colorLED struct {
	r, g, b uint8
}

var (
	off    = colorLED{0, 0, 0}
	red    = colorLED{128, 0, 0}
	green  = colorLED{0, 128, 0}
	blue   = colorLED{0, 0, 128}
	yellow = colorLED{96, 144, 0}
	pink   = colorLED{128, 16, 32}
)

type stmHelper struct {
	stm.Interface
}

// setLED wraps stm's SetLED so that simple color definitions may be used.
func (sh *stmHelper) setLED(led stm.LED, col colorLED) (err error) {
	return sh.SetLED(led, col.r, col.g, col.b)
}

// blinkMaintenanceLED alternates between LED1 and LED2 lighting each
// with yellow color for approximately 1 second.
//
// It is cancelled by ctx.
func (sh *stmHelper) blinkMaintenanceLED(ctx context.Context) {
	defer sh.ClearDisplay()
	for {
		sh.setLED(stm.LED1, yellow)
		time.Sleep(time.Second)
		sh.setLED(stm.LED1, off)

		sh.setLED(stm.LED2, yellow)
		time.Sleep(time.Second)
		sh.setLED(stm.LED2, off)

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// printMessage clears the OLED display and prints msg to it.
func (sh *stmHelper) printMessage(msg string) (err error) {
	err = sh.ClearDisplay()
	if err != nil {
		return err
	}
	return sh.PrintText(0, 0, stm.Foreground, msg)
}

// powerTick switches relay on and off.
// It may be used for Healthcheck.
func (sh *stmHelper) powerTick() (err error) {
	return sh.PowerTick(time.Second)
}
