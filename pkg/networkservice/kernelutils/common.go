// Copyright 2019 SUSE LLC.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kernelutils

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"

  "net"

  "github.com/sirupsen/logrus"
  "github.com/vishvananda/netlink"
)

// Kernel forwarding plane related constants
const (
	cLOCAL    = iota
	cINCOMING = iota
	cOUTGOING = iota
)

const (
	/* VETH pairs are used only for local connections(same node), so we can use a larger MTU size as there's no multi-node connection */
	cVETHMTU    = 16000
	cCONNECT    = true
	cDISCONNECT = false
)

type connectionConfig struct {
	id            string
	srcNetNsInode string
	dstNetNsInode string
	srcName       string
	dstName       string
	srcIP         string
	dstIP         string
	srcIPVXLAN    net.IP
	dstIPVXLAN    net.IP
	srcRoutes     []*connectioncontext.Route
	dstRoutes     []*connectioncontext.Route
	neighbors     []*connectioncontext.IpNeighbor
	vni           int
}

// addNeighbors adds neighbors
func AddNeighbors(link netlink.Link, neighbors []*connectioncontext.IpNeighbor) error {
	for _, neighbor := range neighbors {
		mac, err := net.ParseMAC(neighbor.GetHardwareAddress())
		if err != nil {
			logrus.Error("common: failed parsing the MAC address for IP neighbors:", err)
			return err
		}
		neigh := netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        0x02, // netlink.NUD_REACHABLE, // the constant is somehow not being found in the package in case of using a darwin based machine
			IP:           net.ParseIP(neighbor.GetIp()),
			HardwareAddr: mac,
		}
		if err = netlink.NeighAdd(&neigh); err != nil {
			logrus.Error("common: failed adding neighbor:", err)
			return err
		}
	}
	return nil
}

// newVETH returns a VETH interface instance
func NewVETH(srcName, dstName string) *netlink.Veth {
	/* Populate the VETH interface configuration */
	return &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: srcName,
			MTU:  cVETHMTU,
		},
		PeerName: dstName,
	}
}
