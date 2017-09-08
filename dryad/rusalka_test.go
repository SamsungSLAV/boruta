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

package dryad_test

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	. "git.tizen.org/tools/boruta/dryad"
	"git.tizen.org/tools/muxpi/sw/nanopi/stm"

	"git.tizen.org/tools/boruta"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rusalka", func() {
	var d boruta.Dryad
	const (
		username           = "test-user"
		homeDir            = "/home/" + username
		sshDir             = homeDir + "/.ssh/"
		authorizedKeysFile = sshDir + "authorized_keys"
	)

	BeforeEach(func() {
		err := stm.Open()
		if err != nil {
			Skip(fmt.Sprintf("STM is probably missing: %s", err))
		}
		err = stm.Close()
		Expect(err).ToNot(HaveOccurred())
		d = NewRusalka(username, []string{"users"})
	})

	It("should put in maintenance", func() {
		err := d.PutInMaintenance("test message")
		Expect(err).ToNot(HaveOccurred())
		// TODO(amistewicz): somehow check that goroutine is running and can be terminated.
		time.Sleep(10 * time.Second)
	})

	It("should prepare", func() {
		key, err := d.Prepare()
		Expect(err).ToNot(HaveOccurred())
		Expect(sshDir).To(BeADirectory())
		Expect(authorizedKeysFile).To(BeARegularFile())
		// TODO(amistewicz): Test for file permissions
		// sshDir should have 0755 boruta-user
		// authorizedKeysFile should be owned by boruta-user
		pem.Encode(os.Stderr, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})
	})

	It("should be healthy", func() {
		err := d.Healthcheck()
		Expect(err).ToNot(HaveOccurred())
	})
})
