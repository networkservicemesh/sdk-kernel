// Copyright (c) 2021 Nordix Foundation.
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

package inject

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
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

func renameInterface(origIfName, desiredIfName string, curNetNS, targetNetNS netns.NsHandle) error {
	return nshandle.RunIn(curNetNS, targetNetNS, func() error {
		link, err := netlink.LinkByName(origIfName)
		if err != nil {
			return errors.Wrapf(err, "failed to get net interface: %v", origIfName)
		}

		if err = netlink.LinkSetName(link, desiredIfName); err != nil {
			return errors.Wrapf(err, "failed to rename net interface: %v -> %v", origIfName, desiredIfName)
		}
		return nil
	})
}

func moveAndRename(ctx context.Context, logger log.Logger, conn *networkservice.Connection, isMoveBack bool) error {
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

	vfConfig := vfconfig.Config(ctx)
	ifName := mech.GetInterfaceName()
	if !isMoveBack {
		if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
			err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, curNetNS, curNetNS, targetNetNS)
			if err == nil {
				err = renameInterface(vfConfig.VFInterfaceName, ifName, curNetNS, targetNetNS)
			}
		} else {
			err = moveInterfaceToAnotherNamespace(ifName, curNetNS, curNetNS, targetNetNS)
		}
	} else {
		if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
			err = renameInterface(ifName, vfConfig.VFInterfaceName, curNetNS, targetNetNS)
			if err == nil {
				err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, curNetNS, targetNetNS, curNetNS)
			}
		} else {
			err = moveInterfaceToAnotherNamespace(ifName, curNetNS, targetNetNS, curNetNS)
		}
	}
	if err != nil {
		logger.Warnf("failed to move network interface %s into the target namespace for connection %s", ifName, conn.GetId())
		return err
	}
	logger.Infof("moved network interface %s into the target namespace for connection %s", ifName, conn.GetId())
	return nil
}
