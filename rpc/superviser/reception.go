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

// Package superviser provides Go RPC implementation of client and server for Superviser interface.
//
// It also provides superviserReception that may be used with StartSuperviserReception to record IP
// address of the client and call SetWorkerIP.
package superviser

import (
	"net"
	"net/rpc"

	"git.tizen.org/tools/boruta"
	"git.tizen.org/tools/boruta/workers"
)

type superviserReception struct {
	wl       *workers.WorkerList
	listener net.Listener
}

type addressBook struct {
	ip net.IP
	wl *workers.WorkerList
}

func startSuperviserReception(wl *workers.WorkerList, addr string) (sr *superviserReception, err error) {
	sr = new(superviserReception)
	sr.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return
	}
	sr.wl = wl
	go sr.listenAndServe()
	return
}

// StartSuperviserReception starts listener on addr. For each connection it extracts information
// about client's IP address and serves the connection. In the handler of Register, the extracted
// address is used in call to SetWorkerIP of WorkerList after successful Register of WorkerList.
//
// SetFail is unchanged, i.e. it calls SetFail of WorkerList without modification of arguments and
// return values.
func StartSuperviserReception(wl *workers.WorkerList, addr string) (err error) {
	_, err = startSuperviserReception(wl, addr)
	return err
}

func (sr *superviserReception) listenAndServe() {
	for {
		conn, err := sr.listener.Accept()
		if err != nil {
			// FIXME(amistewicz): properly handle the error as a busy loop is possible.
			continue
		}
		go sr.serve(conn)
	}
}

// serve extracts IP address of the client and stores it in a newly created instance of
// connIntercepter, which will use it for SetWorkerIP call.
func (sr *superviserReception) serve(conn net.Conn) {
	ip := conn.RemoteAddr().(*net.TCPAddr).IP

	sub := &addressBook{
		ip: ip,
		wl: sr.wl,
	}

	srv := rpc.NewServer()
	err := RegisterSuperviserService(srv, sub)
	if err != nil {
		// TODO(amistewicz): log an error.
		return
	}

	srv.ServeConn(conn)
}

// Register calls Register and SetWorkerIP of WorkerList if the former call was successful.
func (ab *addressBook) Register(caps boruta.Capabilities) (err error) {
	err = ab.wl.Register(caps)
	if err != nil {
		return
	}
	return ab.wl.SetWorkerIP(caps.GetWorkerUUID(), ab.ip)
}

// SetFail calls SetFail of WorkerList.
func (ab *addressBook) SetFail(uuid boruta.WorkerUUID, reason string) (err error) {
	return ab.wl.SetFail(uuid, reason)
}