// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
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

package ipaddress

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		// Note: These are switched from normal because if we are the client, we need to assign the IP
		// in the Endpoints NetNS for the Dst.  If we are the *server* we need to assign the IP for the
		// clients NetNS (ie the source).
		ipNets := conn.GetContext().GetIpContext().GetSrcIPNets()
		if isClient {
			ipNets = conn.GetContext().GetIpContext().GetDstIPNets()
		}
		if ipNets == nil {
			return nil
		}

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

		if err = netlinkHandle.LinkSetUp(l); err != nil {
			return errors.WithStack(err)
		}

		var forwarderNetNS netns.NsHandle
		forwarderNetNS, err = nshandle.Current()
		if err != nil {
			return errors.WithStack(err)
		}
		defer func() { _ = forwarderNetNS.Close() }()

		var targetNetNS netns.NsHandle
		targetNetNS, err = nshandle.FromURL(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer func() { _ = targetNetNS.Close() }()

		disableIPv6Filename := fmt.Sprintf("/proc/sys/net/ipv6/conf/%s/disable_ipv6", l.Attrs().Name)
		if err = nshandle.RunIn(forwarderNetNS, targetNetNS, func() error {
			return ioutil.WriteFile(disableIPv6Filename, []byte("0"), 0600)
		}); err != nil {
			return errors.Wrapf(err, "failed to set %s = 0", disableIPv6Filename)
		}

		ch := make(chan netlink.AddrUpdate)
		done := make(chan struct{})

		if err := netlink.AddrSubscribeAt(targetNetNS, ch, done); err != nil {
			return errors.Wrapf(err, "failed to subscribe for interface address updates")
		}

		defer func() {
			close(done)
			// `ch` should be fully read after the `done` close to prevent goroutine leak in `netlink.AddrSubscribeAt`
			go func() {
				for range ch {
				}
			}()
		}()

		for _, ipNet := range ipNets {
			now := time.Now()
			addr := &netlink.Addr{
				IPNet: ipNet,
				Flags: unix.IFA_F_PERMANENT,
			}
			// Turns out IPv6 uses Duplicate Address Detection (DAD) which
			// we don't need here and which can cause it to take more than a second
			// before anything *works* (even though the interface is up).  This causes
			// cryptic error messages.  To avoid, we use the flag to disable DAD for
			// any IPv6 addresses. Further, it seems that this is only needed for veth type, not if we have a tapv2
			if ipNet != nil && ipNet.IP.To4() == nil {
				addr.Flags |= unix.IFA_F_NODAD
			}
			if err := netlinkHandle.AddrReplace(l, addr); err != nil {
				return errors.Wrapf(err, "attempting to add ip address %s to %s (type: %s) with flags 0x%x", addr.IPNet, l.Attrs().Name, l.Type(), addr.Flags)
			}
			log.FromContext(ctx).
				WithField("link.Name", l.Attrs().Name).
				WithField("Addr", ipNet.String()).
				WithField("duration", time.Since(now)).
				WithField("netlink", "AddrAdd").Debug("completed")
		}
		return waitForIPNets(ctx, ch, l, ipNets)
	}
	return nil
}

func waitForIPNets(ctx context.Context, ch chan netlink.AddrUpdate, l netlink.Link, ipNets []*net.IPNet) error {
	now := time.Now()
	for {
		j := -1
		select {
		case <-ctx.Done():
			return errors.Wrapf(ctx.Err(), "timeout waiting for update to add ip addresses %s to %s (type: %s)", ipNets, l.Attrs().Name, l.Type())
		case update, ok := <-ch:
			if !ok {
				return errors.Errorf("failed to receive update to add ip addresses %s to %s (type: %s)", ipNets, l.Attrs().Name, l.Type())
			}
			if update.LinkIndex == l.Attrs().Index {
				for i := range ipNets {
					if update.LinkAddress.IP.Equal(ipNets[i].IP) && update.Flags&unix.IFA_F_TENTATIVE == 0 {
						j = i
						log.FromContext(ctx).
							WithField("AddrUpdate.LinkAddress", update.LinkAddress).
							WithField("link.Name", l.Attrs().Name).
							WithField("duration", time.Since(now)).
							Debug("complete")
						break
					}
				}
			}
		}
		if j != -1 {
			ipNets = append(ipNets[:j], ipNets[j+1:]...)
		}
		if len(ipNets) == 0 {
			return nil
		}
	}
}
