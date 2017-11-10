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

// Package http provides datatypes that are shared between server and client.
package http

import (
	"net"

	. "git.tizen.org/tools/boruta"
)

// ReqIDPack is used for JSON (un)marshaller.
type ReqIDPack struct {
	ReqID
}

// WorkerStatePack is used by JSON (un)marshaller.
type WorkerStatePack struct {
	WorkerState
}

// AccessInfo2 structure is used by HTTP instead of AccessInfo when acquiring
// worker. The only difference is that key field is in PEM format instead of
// rsa.PrivateKey. It is temporary solution - session private keys will be
// replaces with users' public keys when proper user support is added.
type AccessInfo2 struct {
	// Addr is necessary information to connect to a tunnel to Dryad.
	Addr *net.TCPAddr
	// Key is private RSA key in PEM format.
	Key string
	// Username is a login name for the job session.
	Username string
}
