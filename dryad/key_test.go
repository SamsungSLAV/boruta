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
	"crypto/rand"
	"crypto/rsa"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("key generator", func() {
	generate := func(size int) {
		_, err := rsa.GenerateKey(rand.Reader, size)
		Expect(err).ToNot(HaveOccurred())
	}

	Measure("2048 should be fast", func(b Benchmarker) {
		for _, size := range []int{512, 1024, 2048, 4096} {
			b.Time(strconv.Itoa(size), func() {
				generate(size)
			})
		}
	}, 2)
})
