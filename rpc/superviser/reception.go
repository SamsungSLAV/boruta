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

//go:generate go-rpcgen --source=../../boruta.go --type=Superviser --target=superviser.go --package=superviser --imports net/rpc,.=github.com/SamsungSLAV/boruta

import (
	"errors"
	"net"
	"net/rpc"

	"github.com/SamsungSLAV/boruta"
	// Needed for SSH public key serialization.
	_ "github.com/SamsungSLAV/boruta/rpc/types"
)

type superviserReception struct {
	sv       boruta.Superviser
	listener net.Listener
}

type addressBook struct {
	ip net.IP
	sv boruta.Superviser
}

func startSuperviserReception(sv boruta.Superviser, addr string) (sr *superviserReception, err error) {
	sr = new(superviserReception)
	sr.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return
	}
	sr.sv = sv
	go sr.listenAndServe()
	return
}

// StartSuperviserReception starts listener on addr. For each connection it extracts information
// about client's IP address and serves the connection. In the handler of Register, the extracted
// address is used in call to SetWorkerIP of WorkerList after successful Register of WorkerList.
//
// SetFail is unchanged, i.e. it calls SetFail of WorkerList without modification of arguments and
// return values.
func StartSuperviserReception(sv boruta.Superviser, addr string) (err error) {
	_, err = startSuperviserReception(sv, addr)
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
		sv: sr.sv,
	}

	srv := rpc.NewServer()
	err := RegisterSuperviserService(srv, sub)
	if err != nil {
		// TODO(amistewicz): log an error.
		return
	}

	srv.ServeConn(conn)
}

func (ab *addressBook) getTCPAddr(str, ctx string) (*net.TCPAddr, error) {
	if str == "" {
		return nil, errors.New(ctx + " can't be empty")
	}
	return net.ResolveTCPAddr("tcp", str)
}

// Register calls Register of WorkerList. It additionally fills dryadAddress
// and sshAddress with IP address if one is missing from parameters.
func (ab *addressBook) Register(caps boruta.Capabilities, dryadAddress string, sshAddress string) (err error) {
	dryad, err := ab.getTCPAddr(dryadAddress, "dryadAddress")
	if err != nil {
		return err
	}
	if dryad.IP == nil {
		dryad.IP = ab.ip
	}
	sshd, err := ab.getTCPAddr(sshAddress, "sshAddress")
	if err != nil {
		return err
	}
	if sshd.IP == nil {
		sshd.IP = ab.ip
	}
	return ab.sv.Register(caps, dryad.String(), sshd.String())
}

// SetFail calls SetFail of WorkerList.
func (ab *addressBook) SetFail(uuid boruta.WorkerUUID, reason string) (err error) {
	return ab.sv.SetFail(uuid, reason)
}
