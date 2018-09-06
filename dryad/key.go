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

	"git.tizen.org/tools/slav/logger"
)

// installPublicKey marshals and stores key in a proper location to be read by ssh daemon.
func installPublicKey(key *ssh.PublicKey, homedir, uid, gid string) error {
	if key == nil {
		logger.Error("Empty public ssh key provided.")
		return errors.New("empty public key")
	}
	sshDir := path.Join(homedir, ".ssh")
	err := os.MkdirAll(sshDir, 0755)
	if err != nil {
		logger.Error("Failed to create ssh directory.")
		return err
	}
	f, err := os.OpenFile(path.Join(sshDir, "authorized_keys"),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logger.Errorf("Failed to open %s/authorized_keys file.", sshDir)
		return err
	}
	defer f.Close()
	err = updateOwnership(f, sshDir, uid, gid)
	if err != nil {
		logger.Error("Failed to update ownership of authorized_keys file.")
		return err
	}
	_, err = f.Write(ssh.MarshalAuthorizedKey(*key))
	if err != nil {
		logger.Errorf("ssh key: %v failed to marshal: %v", key, err)
	}
	logger.WithProperty("method", "installPublicKey").
		WithProperty("public-key", key).
		Debug("Public Key Installed")
	return err
}

// updateOwnership changes the owner of key and sshDir to uid:gid parsed from uidStr and gidStr.
func updateOwnership(key *os.File, sshDir, uidStr, gidStr string) (err error) {
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		logger.Errorf("%s is not valid UID: %v", uidStr, err)
		return
	}
	gid, err := strconv.Atoi(gidStr)
	if err != nil {
		logger.Errorf("%s is not valid UID: %v", gidStr, err)
		return
	}
	err = os.Chown(sshDir, uid, gid)
	if err != nil {
		logger.Errorf("%s failed to chown for UID: %s GID: %s, due: %v",
			sshDir, uid, gid, err)
		return
	}
	if err = key.Chown(uid, gid); err != nil {
		logger.Errorf("%v key failed to chown for UID: %s, GID: %s, due: %v",
			key, uidStr, gidStr, err)
	}
	logger.WithProperty("ssh-key", key.Name()).WithProperty("ssh-directory", sshDir).
		Info("Updated ownership of ssh directory and key")
	return
}
