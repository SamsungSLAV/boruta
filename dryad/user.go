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
	"fmt"
	"os/exec"
	"os/user"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
)

var (
	userAddExit = map[int]string{
		0:  "success",
		1:  "can't update password file",
		2:  "invalid command syntax",
		3:  "invalid argument to option",
		4:  "UID already in use (and no -o)",
		6:  "specified group doesn't exist",
		9:  "username already in use",
		10: "can't update group file",
		12: "can't create home directory",
		14: "can't update SELinux user mapping",
	}
	userDelExit = map[int]string{
		0:  "success",
		1:  "can't update password file",
		2:  "invalid command syntax",
		6:  "specified user doesn't exist",
		8:  "user currently logged in",
		10: "can't update group file",
		12: "can't remove home directory",
	}
)

// borutaUser is a representation of the local Unix user on Dryad.
type borutaUser struct {
	username string
	groups   []string
	user     *user.User
}

func newBorutaUser(username string, groups []string) *borutaUser {
	return &borutaUser{
		username: username,
		groups:   groups,
	}
}

func prepareGroups(groups []string) string {
	return strings.Join(groups, ",")
}

// handleUserCmdError attempts to use predefined error message. Otherwise err and output are used.
// It works for useradd and userdel exit codes.
func handleUserCmdError(codeToMsg map[int]string, output []byte, err error) error {
	// On Unix systems check exit code and replace error message with predefined string.
	if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			if errmsg, ok := codeToMsg[status.ExitStatus()]; ok {
				return fmt.Errorf("command failed: %s", errmsg)
			}
		}
	}
	return fmt.Errorf("command failed: %s, %s", err, string(output))
}

// prepareUserAddCmd is a helper function for add().
func prepareUserAddCmd(username string, groups []string) *exec.Cmd {
	// "-p" encrypted password of the new account.
	// "-m" create the user's home directory.
	// "-G" list of supplementary groups of the new account.
	return exec.Command("useradd", "-p", "*", "-m", "-G", prepareGroups(groups), username)
}

// add creates borutaUser on the system.
// It does nothing if the user with given username already exists.
func (bu *borutaUser) add() error {
	if bu.update() == nil {
		// user already exists.
		return nil
	}
	output, err := prepareUserAddCmd(bu.username, bu.groups).CombinedOutput()
	if err != nil {
		return handleUserCmdError(userAddExit, output, err)
	}
	return nil
}

// prepareUserDelCmd is a helper function for delete().
func prepareUserDelCmd(username string) *exec.Cmd {
	// "-r" remove home directory and mail spool.
	// "-f" force removal of files, even if not owned by user.
	return exec.Command("userdel", "-r", "-f", username)
}

// delete removes borutaUser from the system. It does nothing if the user with given username
// has already been removed. It invalidates user field of borutaUser.
func (bu *borutaUser) delete() error {
	err := bu.update()
	if err != nil {
		if _, ok := err.(user.UnknownUserError); ok {
			// user already does not exist.
			return nil
		}
		return err
	}
	output, err := prepareUserDelCmd(bu.username).CombinedOutput()
	if err != nil {
		return handleUserCmdError(userDelExit, output, err)
	}
	bu.user = nil
	return nil
}

// update retrieves information about borutaUser from the system. It should be used after userAdd.
// Stored information may be invalid after call to userdel.
func (bu *borutaUser) update() (err error) {
	bu.user, err = user.Lookup(bu.username)
	return
}

// installKey calls installPublicKey with parameters retrieved from the user field
// of borutaUser structure. This filed must be set before call to this function by update() method.
func (bu *borutaUser) installKey(key *ssh.PublicKey) error {
	return installPublicKey(key, bu.user.HomeDir, bu.user.Uid, bu.user.Gid)
}
