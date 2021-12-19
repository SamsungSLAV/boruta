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

// Package types contains gob registration of types needed for RPC
// communication between dryad and supervisor.
package types

import (
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/gob"

	"golang.org/x/crypto/ssh"
)

func init() {
	var rsaKey rsa.PublicKey
	ed25519Key, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic("this should never happen: generating ed25519 key failed: " + err.Error())
	}
	for _, foreignKey := range []interface{}{&rsaKey, ed25519Key} {
		sshKey, err := ssh.NewPublicKey(foreignKey)
		if err != nil {
			panic("this should never happen: arbitrary ssh key preparation failed: " + err.Error())
		}
		gob.Register(sshKey)
	}
}
