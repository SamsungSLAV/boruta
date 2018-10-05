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

package client_test

import (
	"log"
	"os"
	"time"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/http/client"
)

func Example() {
	cl := client.NewBorutaClient("http://localhost:1234")

	caps := make(boruta.Capabilities)
	caps["arch"] = "armv7l"
	validAfter := time.Now()
	deadline := validAfter.Add(time.Hour)

	// Create new Boruta request.
	id, err := cl.NewRequest(caps, boruta.Priority(4), boruta.UserInfo{},
		validAfter, deadline)
	if err != nil {
		log.Fatalln("unable to create new request:", err)
	}

	// Check state of created request.
	state, err := cl.GetRequestState(id)
	if err != nil {
		log.Fatalln("unable to check state of request:", err)
	}
	if state == boruta.INPROGRESS {
		// Acquire worker if the request is in "IN PROGRESS" state.
		access, err := cl.AcquireWorker(id)
		if err != nil {
			log.Fatalln("unable to acquire worker:", err)
		}
		log.Println("dryad address:", access.Addr)
		log.Println("dryad username:", access.Username)
		// Connect to dryad using access variable (boruta.AccessInfo type).
		// ...
		timeout, err := cl.GetJobTimeout(id)
		if err != nil {
			log.Fatalln("unable to check timeout of a job:", err)
		}
		log.Println("job will timeout on", timeout)

		select {
		case <-time.After(timeout.Sub(time.Now())):
			log.Fatalln("job timeout passed, prolong access next time")
		default:
			// Do stuff.
			// ...

			// Close request after stuff was done.
			if err = cl.CloseRequest(id); err != nil {
				log.Fatalln("unable to close request", err)
			}
			log.Println("closed request", id)
			os.Exit(0)
		}
	}
	log.Printf("request %d in state %s\n", id, state)
}
