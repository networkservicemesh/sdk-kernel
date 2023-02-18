// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package ethernetcontext

import (
	"bytes"
	"context"
	"net"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

func setKernelHwAddress(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil {
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

		if ethernetContext := conn.GetContext().GetEthernetContext(); ethernetContext != nil {
			var macAddrString string
			if isClient {
				macAddrString = ethernetContext.GetDstMac()
			} else {
				macAddrString = ethernetContext.GetSrcMac()
			}

			if macAddrString != "" {
				now := time.Now()
				var macAddr net.HardwareAddr
				macAddr, err := net.ParseMAC(macAddrString)
				if err != nil {
					return errors.Wrapf(err, "invalid MAC address: %v", macAddrString)
				}
				if bytes.Equal([]byte(macAddr), []byte(l.Attrs().HardwareAddr)) {
					return nil
				}
				if err = netlinkHandle.LinkSetDown(l); err != nil {
					return errors.Wrapf(err, "failed to disable link device %s", l.Attrs().Name)
				}
				if err = netlinkHandle.LinkSetHardwareAddr(l, macAddr); err != nil {
					return errors.Wrapf(err, "failed to set MAC address for the VF: %v", macAddr)
				}
				if err = netlinkHandle.LinkSetUp(l); err != nil {
					return errors.Wrapf(err, "failed to setup link for the interface %v", l)
				}
				log.FromContext(ctx).
					WithField("link.Name", l.Attrs().Name).
					WithField("MACAddr", macAddrString).
					WithField("duration", time.Since(now)).
					WithField("netlink", "LinkSetHardwareAddr").Debug("completed")
			}
		}
	}
	return nil
}

func vfCreate(vfConfig *vfconfig.VFConfig, conn *networkservice.Connection, isClient bool) error {
	pfLink, err := netlink.LinkByName(vfConfig.PFInterfaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to get PF network interface: %v", vfConfig.PFInterfaceName)
	}

	if ethernetContext := conn.GetContext().GetEthernetContext(); ethernetContext != nil {
		var macAddrString string
		if isClient {
			macAddrString = ethernetContext.GetDstMac()
		} else {
			macAddrString = ethernetContext.GetSrcMac()
		}
		if macAddrString != "" {
			var macAddr net.HardwareAddr
			macAddr, err = net.ParseMAC(macAddrString)
			if err != nil {
				return errors.Wrapf(err, "invalid MAC address: %v", macAddrString)
			}
			if err = netlink.LinkSetVfHardwareAddr(pfLink, vfConfig.VFNum, macAddr); err != nil {
				return errors.Wrapf(err, "failed to set MAC address for the VF: %v", macAddr)
			}
		}
		if vlanTag := int(ethernetContext.GetVlanTag()); vlanTag != 0 {
			if err = netlink.LinkSetVfVlan(pfLink, vfConfig.VFNum, vlanTag); err != nil {
				return errors.Wrapf(err, "failed to set VLAN for the VF: %v", vlanTag)
			}
		}
	}

	return nil
}
