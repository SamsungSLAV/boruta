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

// Package tunnels allows creation of simple forwarding tunnels
// between address pairs.
package tunnels

import (
	"io"
	"net"
)

// Tunnel forwards data between source and destination addresses.
type Tunnel struct {
	Tunneler
	listener *net.TCPListener
	dest     *net.TCPAddr
	done     chan struct{}
}

// Create sets up data forwarding tunnel between src and dest addresses.
// It will listen on random port on src and forward to dest.
//
// When connection to src is made a corresponding one is created to dest
// and data is copied between them.
//
// Close should be called to clean up this function and terminate connections.
func (t *Tunnel) Create(src net.IP, dest net.TCPAddr) (err error) {
	t.dest = &dest
	t.done = make(chan struct{})
	// It will listen on a random port.
	t.listener, err = net.ListenTCP("tcp", &net.TCPAddr{IP: src})
	if err != nil {
		return err
	}
	go t.listenAndForward()
	return nil
}

// Close stops listening on the port for new connections and terminates exisiting ones.
func (t *Tunnel) Close() error {
	close(t.done)
	return t.listener.Close()
}

// listenAndForward accepts connection, creates a corresponding one to dest
// and starts goroutines copying data in both directions.
//
// Connection close does not guarantee that the other one is terminated.
// It does however stop copying data from the closed end.
func (t *Tunnel) listenAndForward() {
	for {
		select {
		case <-t.done:
			// Stop listening if the tunnel is no longer active.
			return
		default:
		}
		srcConn, err := t.listener.Accept()
		if err != nil {
			// TODO(amistewicz): log an error.
			continue
		}
		destConn, err := net.DialTCP("tcp", nil, t.dest)
		if err != nil {
			// TODO(amistewicz): log an error.
			srcConn.Close()
			continue
		}
		// Close connections when we are done.
		defer srcConn.Close()
		defer destConn.Close()
		// Forward traffic in both directions.
		// These goroutines will stop when Close() is called on srcConn and destConn.
		go t.forward(srcConn, destConn)
		go t.forward(destConn, srcConn)
	}
}

// Addr returns address on which it listens for connection.
//
// It should be used to make a connection to the Tunnel.
func (t *Tunnel) Addr() net.Addr {
	return t.listener.Addr()
}

// forward copies data from src to dest.
//
// It is the simplest and most portable solution.
// Properly an iptables entries should be made as follows:
// * -t nat -A PREROUTING -i src_interface -p tcp --dport src_port -j DNAT --to-destination dest_address
// * -A FORWARD -i src_interface -o dest_interface -p tcp --syn --dport src_port -m conntrack --ctstate NEW -j ACCEPT
// * -A FORWARD -i src_interface -o dest_interface -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
// * -A FORWARD -i dest_interface -o src_interface -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
// * -t nat -A POSTROUTING -o dest_interface -p tcp --dport dest_port -d dest_ip -j SNAT --to-source src_ip
func (t *Tunnel) forward(src io.Reader, dest io.Writer) {
	_, err := io.Copy(dest, src)
	// TODO(amistewicz): save statistics about usage in each direction.
	if err != nil {
		// TODO(amistewicz): log an error. It will occur every time a dest end is closed before src.
	}
}
