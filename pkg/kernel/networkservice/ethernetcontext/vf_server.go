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
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type vfEthernetContextServer struct{}

// NewVFServer returns a new VF ethernet context server chain element
func NewVFServer() networkservice.NetworkServiceServer {
	return &vfEthernetContextServer{}
}

func (s *vfEthernetContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if vfConfig := vfconfig.Config(ctx); vfConfig != nil {
		pfLink, err := netlink.LinkByName(vfConfig.PFInterfaceName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get PF network interface: %v", vfConfig.PFInterfaceName)
		}

		if ethernetContext := request.GetConnection().GetContext().GetEthernetContext(); ethernetContext != nil {
			if macAddrString := ethernetContext.GetSrcMac(); macAddrString != "" {
				var macAddr net.HardwareAddr
				macAddr, err = net.ParseMAC(ethernetContext.GetSrcMac())
				if err != nil {
					return nil, errors.Wrapf(err, "invalid MAC address: %v", ethernetContext.GetSrcMac())
				}
				if err = netlink.LinkSetVfHardwareAddr(pfLink, vfConfig.VFNum, macAddr); err != nil {
					return nil, errors.Wrapf(err, "failed to set MAC address for the VF: %v", macAddr)
				}
			}

			if vlanTag := int(ethernetContext.GetVlanTag()); vlanTag != 0 {
				if err = netlink.LinkSetVfVlan(pfLink, vfConfig.VFNum, vlanTag); err != nil {
					return nil, errors.Wrapf(err, "failed to set VLAN for the VF: %v", vlanTag)
				}
			}
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *vfEthernetContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
