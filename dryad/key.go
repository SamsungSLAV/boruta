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
	"errors"
	"os"
	"path"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/SamsungSLAV/slav/logger"
)

// installPublicKey marshals and stores key in a proper location to be read by ssh daemon.
func installPublicKey(key *ssh.PublicKey, homedir, uid, gid string) error {
	logger.WithPropertes(logger.Properties{
		"PublicKey": key, "homedir": homedir, "uid": uid, "gid": gid}).
		Debug("Installing Public Key.")
	if key == nil {
		logger.WithProperty("PublicKey", key).Error("Empty public ssh key provided.")
		return errors.New("empty public key")
	}
	sshDir := path.Join(homedir, ".ssh")
	err := os.MkdirAll(sshDir, 0755)
	if err != nil {
		logger.WithProperty("sshDir", sshDir).Error("Failed to create ssh directory.")
		return err
	}
	f, err := os.OpenFile(path.Join(sshDir, "authorized_keys"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.WithProperty("path", sshDir+"authorized_keys").
			Error("Failed to open authorized_keys file.")
		return err
	}
	defer f.Close()
	err = updateOwnership(f, sshDir, uid, gid)
	if err != nil {
		logger.WithProperty("path", sshDir+"authorized_keys").
			Error("Failed to update ownership of authorized_keys file.")
		return err
	}
	_, err = f.Write(ssh.MarshalAuthorizedKey(*key))
	return err
}

// updateOwnership changes the owner of key and sshDir to uid:gid parsed from uidStr and gidStr.
func updateOwnership(key *os.File, sshDir, uidStr, gidStr string) (err error) {
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		return
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil {
		return
	}
	err = os.Chown(sshDir, uid, gid)
	if err != nil {
		return
	}
	return key.Chown(uid, gid)
}
