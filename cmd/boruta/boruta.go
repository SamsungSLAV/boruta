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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/http/server/api"
	"github.com/SamsungSLAV/boruta/matcher"
	"github.com/SamsungSLAV/boruta/requests"
	"github.com/SamsungSLAV/boruta/rpc/superviser"
	"github.com/SamsungSLAV/boruta/workers"
)

// day is 24 hours in seconds.
const day = 86400

var (
	// origins contains Origins that should be allowed by CORS.
	// TODO: this is default value, make it configurable
	origins = []string{"*"}

	// maxAge contains value that will be used as a 'Access-Control-Max-Age' header value. This
	// is for CORS.
	// TODO: this is default value, make it configurable
	maxAge = day
)

var (
	apiAddr = flag.String("api-addr", ":8487", "ip:port address of REST API server.")
	rpcAddr = flag.String("rpc-addr", ":7175", "ip:port address of Dryad RPC server.")
	version = flag.Bool("version", false, "print Boruta server version and exit.")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Println("boruta version", boruta.Version)
		os.Exit(0)
	}
	w := workers.NewWorkerList()
	r := requests.NewRequestQueue(w, matcher.NewJobsManager(w))
	a := api.NewAPI(r, w, origins, maxAge)
	err := superviser.StartSuperviserReception(w, *rpcAddr)
	if err != nil {
		log.Fatal("RPC register failed:", err)
	}

	log.Fatal(http.ListenAndServe(*apiAddr, a.Router))
}
