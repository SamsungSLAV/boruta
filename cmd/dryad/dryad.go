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
	"net"
	"net/rpc"
	"os"
	"os/signal"

	"git.tizen.org/tools/boruta/dryad"
	"git.tizen.org/tools/boruta/dryad/conf"
	dryad_rpc "git.tizen.org/tools/boruta/rpc/dryad"
	superviser_rpc "git.tizen.org/tools/boruta/rpc/superviser"
	"git.tizen.org/tools/muxpi/sw/nanopi/stm"
	uuid "github.com/satori/go.uuid"
)

var (
	confPath      string
	configuration *conf.General
)

func init() {
	configuration = conf.NewConf()

	flag.StringVar(&confPath, "conf", "/etc/boruta/dryad.conf", "path to the configuration file")
}

func exitOnErr(ctx string, err error) {
	if err != nil {
		log.Fatal(ctx, err)
	}
}

func generateConfFile() {
	f, err := os.Create(confPath)
	exitOnErr("can't create configuration file:", err)
	defer f.Close()

	u, err := uuid.NewV4()
	if err != nil {
		// can't generate UUID so write config without it.
		// TODO: log a warning.
		goto end
	}
	configuration.Caps["UUID"] = u.String()

end:
	exitOnErr("can't generate new configuration:", configuration.Marshal(f))
}

func readConfFile() {
	f, err := os.Open(confPath)
	exitOnErr("can't open configuration file:", err)
	defer f.Close()
	exitOnErr("can't parse configuration:", configuration.Unmarshal(f))
}

func main() {
	flag.Parse()

	// Read configuration.
	_, err := os.Stat(confPath)
	if err != nil {
		if os.IsNotExist(err) {
			generateConfFile()
			log.Fatal("configuration file generated. Please edit it first")
		}
		log.Fatal("can't access file:", err)
	}
	readConfFile()

	var dev stm.InterfaceCloser
	if configuration.STMsocket != "" {
		cl, err := rpc.Dial("unix", configuration.STMsocket)
		exitOnErr("failed to connect to RPC service:", err)

		dev = stm.NewInterfaceClient(cl)
	} else {
		var err error
		dev, err = stm.GetDefaultSTM()
		exitOnErr("failed to connect to STM:", err)
	}
	defer dev.Close()

	rusalka := dryad.NewRusalka(dev, configuration.User.Name, configuration.User.Groups)

	l, err := net.Listen("tcp", configuration.Address)
	exitOnErr("can't listen on port:", err)
	defer l.Close()

	srv := rpc.NewServer()
	err = dryad_rpc.RegisterDryadService(srv, rusalka)
	exitOnErr("can't start RPC service:", err)

	go srv.Accept(l)

	boruta, err := superviser_rpc.DialSuperviserClient(configuration.BorutaAddress)
	exitOnErr("failed to initialize connection to boruta:", err)
	defer boruta.Close()

	err = boruta.Register(configuration.Caps)
	exitOnErr("failed to register to boruta:", err)

	// Wait for interrupt.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
