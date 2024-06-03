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

package conf

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/SamsungSLAV/boruta"
	"github.com/stretchr/testify/assert"
)

type brokenReader struct {
	io.Reader
}

func (*brokenReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("broken reader")
}

var (
	marshaled = `listen_address = ":7175"
boruta_address = ""
ssh_address = ":22"
sdcard = "/dev/sdX"
stm_path = "/run/stm.socket"

[caps]

[user]
  name = "boruta-user"
  groups = []
`
	empty = `listen_address = ""
boruta_address = ""
ssh_address = ""
sdcard = ""
stm_path = ""
`
	unmarshaled = &General{
		Address:   ":7175",
		SSHAdress: ":22",
		Caps:      boruta.Capabilities(map[string]string{}),
		User: &User{
			Name:   "boruta-user",
			Groups: []string{},
		},
		SDcard:    "/dev/sdX",
		STMsocket: "/run/stm.socket",
	}
)

func TestNewConf(t *testing.T) {
	assert.Equal(t, NewConf(), unmarshaled)
}

func TestMarshal(t *testing.T) {
	testCases := [...]struct {
		name string
		conf *General
		str  string
		err  error
	}{
		{name: "valid", conf: NewConf(), str: marshaled, err: nil},
		{name: "empty", conf: new(General), str: empty, err: nil},
		{name: "nil", conf: nil, str: "", err: nil},
	}
	assert := assert.New(t)

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			assert.ErrorIs(test.conf.Marshal(&b), test.err)
			assert.Equal(b.String(), test.str)
		})
	}
}

func TestUnmarshal(t *testing.T) {
	testCases := [...]struct {
		name   string
		conf   *General
		read   io.Reader
		err    error
		panics bool
	}{
		{name: "valid", conf: NewConf(), read: strings.NewReader(marshaled), err: nil},
		{name: "invalid", conf: new(General), read: strings.NewReader(`/4`), err: errors.New(`toml: line 1: expected '.' or '=', but got '/' instead`)},
		{name: "empty", conf: new(General), read: strings.NewReader(empty), err: nil},
		{name: "brokenReader", conf: new(General), read: new(brokenReader), err: errors.New("broken reader")},
		{name: "nil", conf: new(General), read: nil, err: nil, panics: true},
	}
	assert := assert.New(t)

	for _, test := range testCases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var err error
			g := new(General)
			if test.panics {
				assert.Panics(func() { err = g.Unmarshal(test.read) })
			} else {
				assert.NotPanics(func() { err = g.Unmarshal(test.read) })
			}
			if test.err != nil {
				assert.ErrorContains(err, test.err.Error())
			} else {
				assert.NoError(err)
			}
			assert.Equal(g, test.conf)
		})
	}
}
