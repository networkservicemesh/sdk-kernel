// Copyright (c) 2021-2023 Nordix Foundation.
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

package inject

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	kernellink "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

func moveInterfaceToAnotherNamespace(ifName string, fromNetNS, toNetNS netns.NsHandle, logger log.Logger) error {
	handle, err := netlink.NewHandleAt(fromNetNS)
	if err != nil {
		return errors.Wrap(err, "failed to create netlink fromNetNS handle")
	}

	link, err := handle.LinkByName(ifName)
	if err != nil {
		return errors.Wrapf(err, "failed to get net interface: %v", ifName)
	}

	if err := handle.LinkSetNsFd(link, int(toNetNS)); err != nil {
		return errors.Wrapf(err, "failed to move net interface to net NS: %v %v", ifName, toNetNS)
	}
	logger.Debugf("Interface %v moved from netNS %v into the netNS %v", ifName, fromNetNS, toNetNS)
	return nil
}

func renameInterface(origIfName, desiredIfName string, targetNetNS netns.NsHandle, logger log.Logger) error {
	handle, err := netlink.NewHandleAt(targetNetNS)
	if err != nil {
		return errors.Wrap(err, "failed to create netlink targetNetNS handle")
	}

	link, err := handle.LinkByName(origIfName)
	if err != nil {
		return errors.Wrapf(err, "failed to get net interface: %v", origIfName)
	}

	if err = handle.LinkSetDown(link); err != nil {
		return errors.Wrapf(err, "failed to down net interface: %v -> %v", origIfName, desiredIfName)
	}

	if err = handle.LinkSetName(link, desiredIfName); err != nil {
		return errors.Wrapf(err, "failed to rename net interface: %v -> %v", origIfName, desiredIfName)
	}
	logger.Debugf("Interface renamed %v -> %v in netNS %v", origIfName, desiredIfName, targetNetNS)
	return nil
}

func upInterface(ifName string, targetNetNS netns.NsHandle, logger log.Logger) error {
	handle, err := netlink.NewHandleAt(targetNetNS)
	if err != nil {
		return errors.Wrap(err, "failed to create netlink NS handle")
	}

	link, err := handle.LinkByName(ifName)
	if err != nil {
		return errors.Wrapf(err, "failed to get net interface: %v", ifName)
	}

	if err = handle.LinkSetUp(link); err != nil {
		return errors.Wrapf(err, "failed to up net interface: %v", ifName)
	}
	logger.Debugf("Administrative state for interface %v is set UP in netNS %v", ifName, targetNetNS)
	return nil
}

func deleteInterface(ifName string, targetNetNS netns.NsHandle, logger log.Logger) error {
	handle, err := netlink.NewHandleAt(targetNetNS)
	if err != nil {
		return errors.Wrap(err, "failed to create netlink NS handle")
	}

	link, err := handle.LinkByName(ifName)
	if err != nil {
		return errors.Wrapf(err, "failed to get net interface: %v", ifName)
	}

	if err = handle.LinkDel(link); err != nil {
		return errors.Wrapf(err, "failed to delete interface: %v", ifName)
	}
	logger.Debugf("Interface %v successfully deleted in netNS %v", ifName, targetNetNS)
	return nil
}

func move(ctx context.Context, conn *networkservice.Connection, vfRefCountMap map[string]int, vfRefCountMutex sync.Locker, isClient, isMoveBack bool) error {
	mech := kernel.ToMechanism(conn.GetMechanism())
	logger := log.FromContext(ctx).WithField("inject", "move")
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
	if contNetNS, err = nshandle.FromURL(mech.GetNetNSURL()); err != nil {
		return err
	}
	if !contNetNS.IsOpen() && isMoveBack {
		contNetNS = vfConfig.ContNetNS
	}

	// keep NSE container's net ns open until connection close is done,.
	// this would properly move back VF into host net namespace even when
	// container is accidentally deleted before close.
	if !isClient || isMoveBack {
		defer func() { _ = contNetNS.Close() }()
	}

	vfRefCountMutex.Lock()
	defer vfRefCountMutex.Unlock()

	vfRefKey := vfConfig.VFPCIAddress
	if vfRefKey == "" {
		vfRefKey = vfConfig.VFInterfaceName
	}

	ifName := mech.GetInterfaceName()
	if !isMoveBack {
		err = moveToContNetNS(vfConfig, vfRefCountMap, vfRefKey, ifName, hostNetNS, contNetNS, logger)
		if err != nil {
			// If we got an error, try to move back the vf to the host namespace
			result := moveToHostNetNS(vfConfig, vfRefCountMap, vfRefKey, ifName, hostNetNS, contNetNS, logger)
			if result != nil {
				logger.Warnf("Failed to move interface %s to netNS %v, and to move it back to netNS %v", vfConfig.VFInterfaceName, hostNetNS, contNetNS)
			}
		} else {
			vfConfig.ContNetNS = contNetNS
		}
	} else {
		err = moveToHostNetNS(vfConfig, vfRefCountMap, vfRefKey, ifName, hostNetNS, contNetNS, logger)
	}
	if err != nil {
		// link may not be available at this stage for cases like veth pair (might be deleted in previous chain element itself)
		// or container would have killed already (example: due to OOM error or kubectl delete)
		if strings.Contains(err.Error(), "Link not found") || strings.Contains(err.Error(), "bad file descriptor") {
			logger.Warnf("Can not find interface, might be deleted already (%v)", err)
			return nil
		}
		return err
	}
	return nil
}

func moveToContNetNS(vfConfig *vfconfig.VFConfig, vfRefCountMap map[string]int, vfRefKey, ifName string, hostNetNS, contNetNS netns.NsHandle, logger log.Logger) (err error) {
	if _, exists := vfRefCountMap[vfRefKey]; !exists {
		vfRefCountMap[vfRefKey] = 1
	} else {
		vfRefCountMap[vfRefKey]++
		logger.Debugf("Reference count increased to %d for vfRefKey %s", vfRefCountMap[vfRefKey], vfRefKey)
		return nil
	}
	link, _ := kernellink.FindHostDevice("", ifName, contNetNS)
	if link != nil {
		if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
			hostLink, _ := kernellink.FindHostDevice(vfConfig.VFPCIAddress, vfConfig.VFInterfaceName, hostNetNS)
			if hostLink != nil { // orphan link may remained from failed connection since no reference counter stored for it
				removeOrphanLink(hostLink.GetName(), ifName, hostNetNS, contNetNS, logger)
			} else { // do nothing
				logger.Debugf("Device %s exist; link (%v) is already in the netNS %v", ifName, link.GetLink(), contNetNS)
				return nil
			}
		} else { // do nothing
			logger.Debugf("Device %s exist; (link %v, netNS %v)", ifName, link.GetLink(), contNetNS)
			return nil
		}
	}
	if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
		err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, hostNetNS, contNetNS, logger)
		if err == nil {
			err = renameInterface(vfConfig.VFInterfaceName, ifName, contNetNS, logger)
			if err == nil {
				err = upInterface(ifName, contNetNS, logger)
			}
		}
	} else {
		err = moveInterfaceToAnotherNamespace(ifName, hostNetNS, contNetNS, logger)
	}
	return err
}

func moveToHostNetNS(vfConfig *vfconfig.VFConfig, vfRefCountMap map[string]int, vfRefKey, ifName string, hostNetNS, contNetNS netns.NsHandle, logger log.Logger) error {
	var refCount int
	if count, exists := vfRefCountMap[vfRefKey]; exists && count > 0 {
		refCount = count - 1
		vfRefCountMap[vfRefKey] = refCount
	} else {
		logger.Debugf("No reference for interface %s", vfRefKey)
		return nil
	}

	if refCount == 0 {
		delete(vfRefCountMap, vfRefKey)
		if vfConfig != nil && vfConfig.VFInterfaceName != ifName {
			link, _ := kernellink.FindHostDevice(vfConfig.VFPCIAddress, vfConfig.VFInterfaceName, hostNetNS)
			if link != nil {
				linkName := link.GetName()
				logger.Debugf("Device %s found in netNS %v", linkName, hostNetNS)
				if linkName != vfConfig.VFInterfaceName {
					if err := netlink.LinkSetName(link.GetLink(), vfConfig.VFInterfaceName); err != nil {
						return errors.Wrapf(err, "failed to rename interface from %s to %s: %v", linkName, vfConfig.VFInterfaceName, err)
					}
					logger.Debugf("Interface renamed %s -> %s in netNS %v", linkName, vfConfig.VFInterfaceName, hostNetNS)
				}
				return nil
			}
			err := renameInterface(ifName, vfConfig.VFInterfaceName, contNetNS, logger)
			if err == nil {
				err = moveInterfaceToAnotherNamespace(vfConfig.VFInterfaceName, contNetNS, hostNetNS, logger)
			}
			return err
		}
		link, _ := kernellink.FindHostDevice("", ifName, hostNetNS)
		if link != nil {
			logger.Debugf("Interface %s found in netNS %v", ifName, hostNetNS)
			return nil
		}
		return moveInterfaceToAnotherNamespace(ifName, contNetNS, hostNetNS, logger)
	}
	return nil
}

func removeOrphanLink(hostIfName, ifName string, hostNetNS, contNetNS netns.NsHandle, logger log.Logger) {
	logger.Debugf("Orphan interface %s found on netNS %s and interface %s still in host netNS %s",
		ifName, contNetNS, hostIfName, hostNetNS)
	tmpIfName := getTempName(contNetNS, hostIfName)
	if err := renameInterface(ifName, tmpIfName, contNetNS, logger); err != nil {
		logger.Warnf("Failed to rename orphan interface %s (%v)", ifName, err)
		tmpIfName = ifName
	}
	orphanIntNetNS := hostNetNS
	if err := moveInterfaceToAnotherNamespace(tmpIfName, contNetNS, hostNetNS, logger); err != nil {
		logger.Warnf("Failed to move orphan interface %s back to host netNS %v (%v)", tmpIfName, hostNetNS, err)
		orphanIntNetNS = contNetNS
	}
	if err := deleteInterface(tmpIfName, orphanIntNetNS, logger); err != nil {
		logger.Warnf("Failed to delete interface %s from netNS %v (%v)", tmpIfName, orphanIntNetNS, err)
	}
}

func getTempName(netNS netns.NsHandle, ifName string) string {
	suffix := ifName[strings.IndexByte(ifName, '-')+1:]
	var s unix.Stat_t
	if err := unix.Fstat(int(netNS), &s); err == nil {
		suffix = fmt.Sprintf("%d", s.Ino)
	}
	name := fmt.Sprintf("tmp-%d-%s", int(netNS), suffix)
	if len(name) > kernel.LinuxIfMaxLength {
		name = name[:kernel.LinuxIfMaxLength]
	}
	return name
}
