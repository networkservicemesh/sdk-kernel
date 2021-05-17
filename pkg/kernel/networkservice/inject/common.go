// Copyright (C) 2021, Nordix Foundation
//
// All rights reserved. This program and the accompanying materials
// are made available under the terms of the Apache License, Version 2.0
// which accompanies this distribution, and is available at
// http://www.apache.org/licenses/LICENSE-2.0

package inject

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func moveInterfaceToAnotherNamespace(ifName string, curNetNS, fromNetNS, toNetNS netns.NsHandle) error {
	return nshandle.RunIn(curNetNS, fromNetNS, func() error {
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			return errors.Wrapf(err, "failed to get net interface: %v", ifName)
		}

		if err := netlink.LinkSetNsFd(link, int(toNetNS)); err != nil {
			return errors.Wrapf(err, "failed to move net interface to net NS: %v %v", ifName, toNetNS)
		}

		return nil
	})
}

func move(logger log.Logger, conn *networkservice.Connection, isMoveBack bool) error {
	mech := kernel.ToMechanism(conn.GetMechanism())
	if mech == nil {
		return nil
	}

	curNetNS, err := nshandle.Current()
	if err != nil {
		return err
	}
	defer func() { _ = curNetNS.Close() }()

	var targetNetNS netns.NsHandle
	targetNetNS, err = nshandle.FromURL(mech.GetNetNSURL())
	if err != nil {
		return err
	}
	defer func() { _ = targetNetNS.Close() }()

	ifName := mech.GetInterfaceName(conn)
	if !isMoveBack {
		err = moveInterfaceToAnotherNamespace(ifName, curNetNS, curNetNS, targetNetNS)
	} else {
		err = moveInterfaceToAnotherNamespace(ifName, curNetNS, targetNetNS, curNetNS)
	}
	if err != nil {
		logger.Warnf("failed to move network interface %s into the target namespace for connection %s", ifName, conn.GetId())
		return err
	}
	logger.Infof("moved network interface %s into the target namespace for connection %s", ifName, conn.GetId())
	return nil
}
