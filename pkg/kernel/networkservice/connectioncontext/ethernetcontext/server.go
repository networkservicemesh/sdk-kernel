// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package ethernetcontext provides chain element for setup link ethernet properties
package ethernetcontext

import (
	"context"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type ethernetContextServer struct{}

// NewServer returns a new ethernet context server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &ethernetContextServer{}
}

func (s *ethernetContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	if ethernetContext := request.GetConnection().GetContext().GetEthernetContext(); ethernetContext != nil {
		var macAddr net.HardwareAddr
		var setMacAddr bool
		if macAddrString := ethernetContext.GetSrcMac(); macAddrString != "" {
			macAddr, err = net.ParseMAC(ethernetContext.GetSrcMac())
			if err != nil {
				return nil, errors.Wrapf(err, "invalid MAC address: %v", ethernetContext.GetSrcMac())
			}
			setMacAddr = true
		}

		vlanTag := int(ethernetContext.GetVlanTag())
		var setVlanTag bool
		if vlanTag != 0 {
			setVlanTag = true
		}

		if vfConfig := vfconfig.Config(ctx); vfConfig != nil {
			if err := configureVirtualFunction(vfConfig, setMacAddr, macAddr, setVlanTag, vlanTag); err != nil {
				return nil, err
			}
		} else if mech := kernel.ToMechanism(request.GetConnection().GetMechanism()); mech != nil {
			ifName := mech.GetInterfaceName(request.GetConnection())
			if err := configureNetInterface(ifName, setMacAddr, macAddr, setVlanTag, vlanTag); err != nil {
				return nil, err
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func configureVirtualFunction(
	vfConfig *vfconfig.VFConfig,
	setMacAddr bool, macAddr net.HardwareAddr,
	setVlanTag bool, vlanTag int,
) error {
	pfLink, err := netlink.LinkByName(vfConfig.PFInterfaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to get PF network interface: %v", vfConfig.PFInterfaceName)
	}

	if setMacAddr {
		if err := netlink.LinkSetVfHardwareAddr(pfLink, vfConfig.VFNum, macAddr); err != nil {
			return errors.Wrapf(err, "failed to set MAC address for the VF: %v", macAddr)
		}
	}

	if setVlanTag {
		if err := netlink.LinkSetVfVlan(pfLink, vfConfig.VFNum, vlanTag); err != nil {
			return errors.Wrapf(err, "failed to set VLAN for the VF: %v", vlanTag)
		}
	}

	return nil
}

func configureNetInterface(
	ifName string,
	setMacAddr bool, macAddr net.HardwareAddr,
	_ /* setVlanTag */ bool, _ /* vlanTag */ int,
) error {
	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return errors.Wrapf(err, "failed to get network interface: %v", ifName)
	}

	if setMacAddr {
		if err := netlink.LinkSetHardwareAddr(link, macAddr); err != nil {
			return errors.Wrapf(err, "failed to set MAC address for the network interface: %v %v", ifName, macAddr)
		}
	}

	// if setVlanTag {
	//     TODO: add VLAN for net interface
	// }

	return nil
}

func (s *ethernetContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
