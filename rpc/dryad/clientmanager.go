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

// File rpc/dryad/clientmanager.go defines ClientManager interface with API
// for managing client RPC calls to Dryad.

package dryad

import (
	"net"

	. "git.tizen.org/tools/boruta"
)

// ClientManager defines API for managing client RPC calls to Dryad.
type ClientManager interface {
	Dryad
	// Create creates a new RPC client.
	Create(net.IP, int) error
	// Close shuts down RPC client connection.
	Close() error
}
