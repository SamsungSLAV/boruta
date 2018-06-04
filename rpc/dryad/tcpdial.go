/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
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

// File rpc/dryad/tcpdial.go contains implementation of creating an RPC
// client inside DryadClient using TCP dial.

package dryad

import (
	"net"
	"net/rpc"
)

// Create sets up new TCP dialled RPC client in DryadClient structure.
// The Create function implements ClientManager interface.
func (_c *DryadClient) Create(addr *net.TCPAddr) error {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}
	_c.client = rpc.NewClient(conn)
	return nil
}
