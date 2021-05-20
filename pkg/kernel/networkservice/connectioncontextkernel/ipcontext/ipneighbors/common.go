// Copyright (c) Nordix Foundation.
//
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

// +build linux

package ipneighbors

import (
	"context"
	"net"
	"os"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	kernelmech "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

func create(ctx context.Context, conn *networkservice.Connection) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
		handle, err := nshandle.ToNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer handle.Delete()

		ifName := mechanism.GetInterfaceName(conn)

		err = nshandle.SetLinkUp(ifName)
		if err != nil {
			return errors.WithStack(err)
		}

		l, err := handle.LinkByName(ifName)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := setIPNeighbors(conn.GetContext().GetIpContext().GetIpNeighbors(), l); err != nil {
			return errors.WithStack(err)
		}

	}
	return nil
}

func setIPNeighbors(ipNeighbours []*networkservice.IpNeighbor, link netlink.Link) error {
	for _, ipNeighbor := range ipNeighbours {
		macAddr, err := net.ParseMAC(ipNeighbor.HardwareAddress)
		if err != nil {
			return errors.Wrapf(err, "invalid neighbor MAC address: %v", ipNeighbor.HardwareAddress)
		}
		if err := netlink.NeighAdd(&netlink.Neigh{
			LinkIndex:    link.Attrs().Index,
			State:        kernelmech.NudReachable,
			IP:           net.ParseIP(ipNeighbor.Ip),
			HardwareAddr: macAddr,
		}); err != nil && !os.IsExist(err) {
			return errors.Wrapf(err, "failed to add IP neighbor: %v", ipNeighbor)
		}
	}
	return nil
}
