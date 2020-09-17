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

// Package inject contains chain element that moves network interface to and from a Client's pod network namespace
package inject

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/utils"
)

type injectServer struct {
}

// NewServer - returns a new networkservice.NetworkServiceServer that moves given network interface into the Client's
// pod network namespace on Request and back to Forwarder's network namespace on Close
func NewServer() networkservice.NetworkServiceServer {
	return &injectServer{}
}

func (a *injectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	connID := request.GetConnection().GetId()
	mech := kernel.ToMechanism(request.GetConnection().GetMechanism())

	nsSwitch, err := utils.NewNSSwitch()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init net NS switch")
	}

	nsSwitch.Lock()
	defer nsSwitch.Unlock()

	defer func() {
		if err = nsSwitch.SwitchByNetNSHandle(nsSwitch.NetNSHandle); err != nil {
			panic(errors.Wrap(err, "failed to switch back to forwarder net NS").Error())
		}
		_ = nsSwitch.Close()
	}()

	clientNetNSHandle, err := utils.GetNSHandleFromInode(mech.GetNetNSInode())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifaceName := mech.GetInterfaceName(request.GetConnection())
	if err = a.moveInterfaceToAnotherNamespace(nsSwitch, ifaceName, nsSwitch.NetNSHandle, clientNetNSHandle); err != nil {
		return nil, errors.Wrapf(err, "Failed to move network interface %s into the Client's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Client's namespace for connection %s", ifaceName, connID)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		errMovingBack := a.moveInterfaceToAnotherNamespace(nsSwitch, ifaceName, clientNetNSHandle, nsSwitch.NetNSHandle)
		if errMovingBack != nil {
			log.Entry(ctx).Infof("Failed to move network interface %s into the Forwarder's namespace for connection %s", ifaceName, connID)
		}
		log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, connID)
	}
	return conn, err
}

func (a *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	mech := kernel.ToMechanism(conn.GetMechanism())

	nsSwitch, err := utils.NewNSSwitch()
	if err != nil {
		return nil, errors.Wrap(err, "failed to init net NS switch")
	}

	nsSwitch.Lock()
	defer nsSwitch.Unlock()

	defer func() {
		if err = nsSwitch.SwitchByNetNSHandle(nsSwitch.NetNSHandle); err != nil {
			panic(errors.Wrap(err, "failed to switch back to forwarder net NS").Error())
		}
		_ = nsSwitch.Close()
	}()

	clientNetNSHandle, err := utils.GetNSHandleFromInode(mech.GetNetNSInode())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifaceName := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if ifaceName == "" {
		return nil, errors.New("Interface name is not found")
	}

	err = a.moveInterfaceToAnotherNamespace(nsSwitch, ifaceName, clientNetNSHandle, nsSwitch.NetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Forwarder's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, conn.GetId())

	return next.Server(ctx).Close(ctx, conn)
}

func (a *injectServer) moveInterfaceToAnotherNamespace(nsSwitch *utils.NSSwitch, ifName string, fromNetNS, toNetNS netns.NsHandle) error {
	if err := nsSwitch.SwitchByNetNSHandle(fromNetNS); err != nil {
		return errors.Wrapf(err, "failed to switch to net NS: %v", fromNetNS)
	}

	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return errors.Wrapf(err, "failed to get net interface: %v", ifName)
	}

	if err := netlink.LinkSetDown(link); err != nil {
		return errors.Wrapf(err, "failed to set net interface down: %v", ifName)
	}

	if err := netlink.LinkSetNsFd(link, int(toNetNS)); err != nil {
		return errors.Wrapf(err, "failed to move net interface to net NS: %v %v", ifName, toNetNS)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return errors.Wrapf(err, "failed to set net interface up: %v", ifName)
	}

	return nil
}
