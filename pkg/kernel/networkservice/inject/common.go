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
	"strings"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	kernellink "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
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

		if err = netlink.LinkSetDown(link); err != nil {
			return errors.Wrapf(err, "failed to rename net interface: %v -> %v", origIfName, desiredIfName)
		}

		if err = netlink.LinkSetName(link, desiredIfName); err != nil {
			return errors.Wrapf(err, "failed to rename net interface: %v -> %v", origIfName, desiredIfName)
		}

		if err = netlink.LinkSetUp(link); err != nil {
			return errors.Wrapf(err, "failed to rename net interface: %v -> %v", origIfName, desiredIfName)
		}

		return nil
	})
}

func move(ctx context.Context, conn *networkservice.Connection, isClient, isMoveBack bool) error {
	mech := kernel.ToMechanism(conn.GetMechanism())
	if mech == nil {
		return nil
	}

	vfConfig, ok := vfconfig.Load(ctx, isClient)
	if !ok {
		return nil
	}

	hostNetNS, err := nshandle.Current()
	if err != nil {
		return err
	}
	defer func() { _ = hostNetNS.Close() }()

	var contNetNS netns.NsHandle
	contNetNS, err = nshandle.FromURL(mech.GetNetNSURL())
	if err != nil {
		return err
	}
	if !contNetNS.IsOpen() && isMoveBack {
		contNetNS = vfConfig.ContNetNS
	}
	defer func() { _ = contNetNS.Close() }()

	ifName := mech.GetInterfaceName()
	if !isMoveBack {
		err = moveToContNetNS(vfConfig, ifName, hostNetNS, contNetNS)
		vfConfig.ContNetNS = contNetNS
	} else {
		err = moveToHostNetNS(vfConfig, ifName, hostNetNS, contNetNS)
	}
	if err != nil {
		// link may not be available at this stage for cases like veth pair (might be deleted in previous chain element itself)
		// or container would have killed already (example: due to OOM error or kubectl delete)
		if strings.Contains(err.Error(), "Link not found") || strings.Contains(err.Error(), "bad file descriptor") {
			return nil
		}
		return err
	}
	return nil
}

func moveToContNetNS(vfConfig *vfconfig.VFConfig, ifName string, hostNetNS, contNetNS netns.NsHandle) (err error) {
	link, _ := kernellink.FindHostDevice("", ifName, contNetNS)
	if link != nil {
		return
	}
	if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
		err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, hostNetNS, hostNetNS, contNetNS)
		if err == nil {
			err = renameInterface(vfConfig.VFInterfaceName, ifName, hostNetNS, contNetNS)
		}
	} else {
		err = moveInterfaceToAnotherNamespace(ifName, hostNetNS, hostNetNS, contNetNS)
	}
	return
}

func moveToHostNetNS(vfConfig *vfconfig.VFConfig, ifName string, hostNetNS, contNetNS netns.NsHandle) (err error) {
	if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
		link, _ := kernellink.FindHostDevice(vfConfig.VFPCIAddress, vfConfig.VFInterfaceName, hostNetNS)
		if link != nil {
			linkName := link.GetName()
			if linkName != vfConfig.VFInterfaceName {
				if err = netlink.LinkSetName(link.GetLink(), vfConfig.VFInterfaceName); err != nil {
					err = errors.Wrapf(err, "failed to rename interface from %s to %s", linkName, vfConfig.VFInterfaceName)
				}
			}
			return
		}
		err = renameInterface(ifName, vfConfig.VFInterfaceName, hostNetNS, contNetNS)
		if err == nil {
			err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, hostNetNS, contNetNS, hostNetNS)
		}
	} else {
		link, _ := kernellink.FindHostDevice("", ifName, hostNetNS)
		if link != nil {
			return nil
		}
		err = moveInterfaceToAnotherNamespace(ifName, hostNetNS, contNetNS, hostNetNS)
	}
	return
}
