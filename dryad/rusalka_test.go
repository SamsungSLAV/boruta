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

//go:generate mockgen -destination=muxpi_mock_test.go -package dryad git.tizen.org/tools/muxpi/sw/nanopi/stm Interface

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"os/user"
	"time"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/muxpi/sw/nanopi/stm"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Rusalka", func() {
	var d boruta.Dryad
	var ctrl *gomock.Controller
	var stmMock *MockInterface
	const (
		username           = "test-user"
		homeDir            = "/home/" + username
		sshDir             = homeDir + "/.ssh/"
		authorizedKeysFile = sshDir + "authorized_keys"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		stmMock = NewMockInterface(ctrl)

		d = NewRusalka(stmMock, username, nil)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should put in maintenance", func(done Done) {
		const msg = "test message"

		gomock.InOrder(
			stmMock.EXPECT().ClearDisplay(),
			stmMock.EXPECT().PrintText(uint(0), uint(0), stm.Foreground, msg),
			stmMock.EXPECT().SetLED(stm.LED1, yellow.r, yellow.g, yellow.b),
			stmMock.EXPECT().SetLED(stm.LED1, off.r, off.g, off.b),
			stmMock.EXPECT().SetLED(stm.LED2, yellow.r, yellow.g, yellow.b),
			stmMock.EXPECT().SetLED(stm.LED2, off.r, off.g, off.b),
			stmMock.EXPECT().ClearDisplay().Do(
				func() { close(done) }),
		)

		err := d.PutInMaintenance(msg)
		Expect(err).ToNot(HaveOccurred())
		d.(*Rusalka).cancelMaintenance()
	}, 4) // blinkMaintenanceLED sleeps twice for 2 seconds each time.

	It("should prepare", func() {
		u, err := user.Current()
		Expect(err).ToNot(HaveOccurred())
		if u.Uid != "0" {
			Skip("must be run as root")
		}

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
		stmMock.EXPECT().PowerTick(time.Second)

		err := d.Healthcheck()
		Expect(err).ToNot(HaveOccurred())
	})
})
