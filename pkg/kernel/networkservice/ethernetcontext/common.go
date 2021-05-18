// Copyright (C) 2021, Nordix Foundation
//
// All rights reserved. This program and the accompanying materials
// are made available under the terms of the Apache License, Version 2.0
// which accompanies this distribution, and is available at
// http://www.apache.org/licenses/LICENSE-2.0

package ethernetcontext

import (
	"net"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

func setupEthernetConfig(vfConfig *vfconfig.VFConfig, conn *networkservice.Connection, isClient bool) error {
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
