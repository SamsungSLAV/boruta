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
	"os/user"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("user management", func() {
	var bu *borutaUser

	BeforeEach(func() {
		bu = &borutaUser{
			username: "test-user",
			groups:   []string{"users"},
		}

		u, err := user.Current()
		Expect(err).ToNot(HaveOccurred())
		if u.Uid != "0" {
			Skip("must be run as root")
		}
	})

	It("should add user", func() {
		err := bu.add()
		Expect(err).ToNot(HaveOccurred())
		Expect("/home/test-user/").To(BeADirectory())

		err = bu.update()
		Expect(err).ToNot(HaveOccurred())
	})

	It("should remove user", func() {
		err := bu.delete()
		Expect(err).ToNot(HaveOccurred())
		Expect("/home/boruta-user/").ToNot(BeADirectory())

		err = bu.update()
		Expect(err).To(HaveOccurred())
	})
})
