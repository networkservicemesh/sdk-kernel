// Copyright (c) 2024 Cisco and/or its affiliates.
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

package ipaddresscheck

import (
	"context"
	"net"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		ipNets := conn.GetContext().GetIpContext().GetSrcIPNets()
		if isClient {
			ipNets = conn.GetContext().GetIpContext().GetDstIPNets()
		}
		if ipNets == nil {
			return nil
		}

		toCheck := make([]*net.IPNet, len(ipNets))
		copy(toCheck, ipNets)

		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return err
		}
		defer netlinkHandle.Close()

		ifName := mechanism.GetInterfaceName()
		l, err := netlinkHandle.LinkByName(ifName)
		if err != nil {
			return errors.Wrapf(err, "failed to find link %s", ifName)
		}

		return checkIPNets(ctx, netlinkHandle, l, toCheck)
	}
	return nil
}

func checkIPNets(ctx context.Context, netlinkHandle *netlink.Handle, l netlink.Link, ipNets []*net.IPNet) error {
	now := time.Now()

	current := make(map[string]struct{})
	for _, ipNet := range ipNets {
		current[ipNet.String()] = struct{}{}
	}

	for {
		time.Sleep(time.Millisecond * 500)
		if len(current) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "timeout waiting for update to add ip addresses %s to %s (type: %s)", ipNets, l.Attrs().Name, l.Type())
		default:
			addrs, err := netlinkHandle.AddrList(l, netlink.FAMILY_ALL)
			if err != nil {
				return errors.Wrapf(err, "failed to get ip addresses for %s", l.Attrs().Name)
			}
			for _, addr := range addrs {
				addrString := addr.IPNet.String()
				if _, ok := current[addrString]; ok {
					delete(current, addrString)
					log.FromContext(ctx).
						WithField("LinkAddress", addr).
						WithField("link.Name", l.Attrs().Name).
						WithField("duration", time.Since(now)).
						Debug("complete")
				}
			}
		}
	}
}
