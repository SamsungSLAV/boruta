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

// File tunnels/tunneler.go defines Tunneler interface with API
// for basic operations on tunnels.

package tunnels

import (
	"net"
)

// Tunneler defines API for basic operations on tunnels.
type Tunneler interface {
	// Create sets up a new tunnel.
	Create(net.IP, net.IP) error
	// Close shuts down tunnel.
	Close() error
	// Addr returns the address of the tunnel to be used by a user
	// for a connection to Dryad.
	Addr() net.Addr
}
