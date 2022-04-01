// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

//go:build linux
// +build linux

package ipneighbors

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer netlinkHandle.Close()

		ifName := mechanism.GetInterfaceName()

		l, err := netlinkHandle.LinkByName(ifName)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := setIPContextNeighbors(ctx, netlinkHandle, conn.GetContext().GetIpContext().GetIpNeighbors(), l); err != nil {
			return errors.WithStack(err)
		}

		// If payload is IP - we need to add additional neighbor
		if conn.GetPayload() == payload.IP {
			peerLink, ok := peer.Load(ctx, isClient)
			if !ok {
				log.FromContext(ctx).Error("Peer link not found")
				return nil
			}
			if peerLink == nil || peerLink.Attrs() == nil || peerLink.Attrs().HardwareAddr == nil {
				panic(fmt.Sprintf("unable to construct peer ip neighbor %+v", peerLink))
			}

			dstNets := conn.GetContext().GetIpContext().GetDstIPNets()
			if isClient {
				dstNets = conn.GetContext().GetIpContext().GetSrcIPNets()
			}

			for _, dstNet := range dstNets {
				if dstNet != nil {
					if err := setPeerNeighbor(ctx, netlinkHandle, l, peerLink, dstNet); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func setIPContextNeighbors(ctx context.Context, handle *netlink.Handle, ipNeighbours []*networkservice.IpNeighbor, netLink netlink.Link) error {
	for _, ipNeighbor := range ipNeighbours {
		macAddr, err := net.ParseMAC(ipNeighbor.HardwareAddress)
		if err != nil {
			return errors.Wrapf(err, "invalid neighbor MAC address: %v", ipNeighbor.HardwareAddress)
		}
		now := time.Now()
		neigh := &netlink.Neigh{
			LinkIndex:    netLink.Attrs().Index,
			State:        link.NudReachable,
			IP:           net.ParseIP(ipNeighbor.Ip),
			HardwareAddr: macAddr,
		}
		if err := handle.NeighSet(neigh); err != nil {
			log.FromContext(ctx).
				WithField("linkIndex", neigh.LinkIndex).
				WithField("ip", neigh.IP).
				WithField("state", neigh.State).
				WithField("hardwareAddr", neigh.HardwareAddr).
				WithField("duration", time.Since(now)).
				WithField("netlink", "NeighSet").Error("setIPNeighbors failed")
			return errors.Wrapf(err, "failed to add IP neighbor: %v", ipNeighbor)
		}
		log.FromContext(ctx).
			WithField("linkIndex", neigh.LinkIndex).
			WithField("ip", neigh.IP).
			WithField("state", neigh.State).
			WithField("hardwareAddr", neigh.HardwareAddr).
			WithField("duration", time.Since(now)).
			WithField("netlink", "NeighSet").Debug("setIPNeighbors completed")
	}
	return nil
}

func setPeerNeighbor(ctx context.Context, handle *netlink.Handle, l, peerLink netlink.Link, dstNet *net.IPNet) error {
	now := time.Now()
	neigh := &netlink.Neigh{
		LinkIndex:    l.Attrs().Index,
		IP:           dstNet.IP,
		State:        netlink.NUD_PERMANENT,
		HardwareAddr: peerLink.Attrs().HardwareAddr,
	}

	if err := handle.NeighSet(neigh); err != nil {
		log.FromContext(ctx).
			WithField("linkIndex", neigh.LinkIndex).
			WithField("ip", neigh.IP).
			WithField("state", neigh.State).
			WithField("hardwareAddr", neigh.HardwareAddr).
			WithField("duration", time.Since(now)).
			WithField("netlink", "NeighSet").Error("setPeerNeighbor failed")
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("linkIndex", neigh.LinkIndex).
		WithField("ip", neigh.IP).
		WithField("state", neigh.State).
		WithField("hardwareAddr", neigh.HardwareAddr).
		WithField("duration", time.Since(now)).
		WithField("netlink", "NeighSet").Debug("setPeerNeighbor completed")
	return nil
}
