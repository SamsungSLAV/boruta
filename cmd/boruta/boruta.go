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
	"log"
	"net/http"

	"github.com/SamsungSLAV/boruta/http/server/api"
	"github.com/SamsungSLAV/boruta/matcher"
	"github.com/SamsungSLAV/boruta/requests"
	"github.com/SamsungSLAV/boruta/rpc/superviser"
	"github.com/SamsungSLAV/boruta/workers"
	"github.com/dimfeld/httptreemux"
)

var (
	apiAddr = flag.String("api-addr", ":8487", "ip:port address of REST API server.")
	rpcAddr = flag.String("rpc-addr", ":7175", "ip:port address of Dryad RPC server.")
)

func main() {
	flag.Parse()
	w := workers.NewWorkerList()
	r := requests.NewRequestQueue(w, matcher.NewJobsManager(w))
	router := httptreemux.New()
	_ = api.NewAPI(router, r, w)
	err := superviser.StartSuperviserReception(w, *rpcAddr)
	if err != nil {
		log.Fatal("RPC register failed:", err)
	}

	log.Fatal(http.ListenAndServe(*apiAddr, router))
}
