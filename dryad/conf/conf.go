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

// Package conf manages Dryad's configuration.
package conf

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

// DefaultRPCPort is a port that should be used as default parameter
// for Dryad's RPC client and server.
const DefaultRPCPort = 7175

// NewConf returns a new instance of General configuration with default values set.
func NewConf() *General {
	return &General{
		Address: fmt.Sprintf(":%d", DefaultRPCPort),
		User: &User{
			Name:   "boruta-user",
			Groups: []string{},
		},
		SDcard: "/dev/sdX",
	}
}

// User is a section in a configuration used for user manipulation.
type User struct {
	// Name is a username of a local account. It should be used to establish SSH session
	// to the system Dryad is running on.
	Name string `toml:"name"`
	// Groups is a list of local Unix groups the username belongs to.
	Groups []string `toml:"groups"`
}

// General is a base struct of configuration.
type General struct {
	// Address is used to listen for connection from Boruta.
	Address string `toml:"listen_address"`
	// User refers information necessary to create the user.
	User *User `toml:"user"`
	// SDcard is a base path to block device of sdcard.
	SDcard string `toml:"sdcard"`
}

// Marshal writes TOML representation of g to w.
func (g *General) Marshal(w io.Writer) error {
	return toml.NewEncoder(w).Encode(g)
}

// Unmarshal reads TOML representation from r and parses it into g.
func (g *General) Unmarshal(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return toml.Unmarshal(b, g)
}
