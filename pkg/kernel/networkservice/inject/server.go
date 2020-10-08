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

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nsswitch"
)

type injectServer struct{}

// NewServer - returns a new networkservice.NetworkServiceServer that moves given network interface into the Client's
// pod network namespace on Request and back to Forwarder's network namespace on Close
func NewServer() networkservice.NetworkServiceServer {
	return &injectServer{}
}

func (s *injectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logEntry := log.Entry(ctx).WithField("injectServer", "Request")

	connID := request.GetConnection().GetId()
	mech := kernel.ToMechanism(request.GetConnection().GetMechanism())

	nsSwitch, err := nsswitch.NewNSSwitch()
	if err != nil {
		return nil, err
	}
	defer func() { _ = nsSwitch.Close() }()

	var clientNetNSHandle netns.NsHandle
	clientNetNSHandle, err = nshandle.FromURL(mech.GetNetNSURL())
	if err != nil {
		return nil, err
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifName := mech.GetInterfaceName(request.GetConnection())
	err = moveInterfaceToAnotherNamespace(nsSwitch, ifName, nsSwitch.NetNSHandle, clientNetNSHandle)
	if err != nil {
		return nil, err
	}
	logEntry.Infof("moved network interface %s into the Client's namespace for connection %s", ifName, connID)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if errMovingBack := moveInterfaceToAnotherNamespace(nsSwitch, ifName, clientNetNSHandle, nsSwitch.NetNSHandle); errMovingBack != nil {
			logEntry.Warnf("failed to move network interface %s into the Forwarder's namespace for connection %s", ifName, connID)
		} else {
			logEntry.Infof("moved network interface %s into the Forwarder's namespace for connection %s", ifName, connID)
		}
	}
	return conn, err
}

func (s *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logEntry := log.Entry(ctx).WithField("injectServer", "Close")

	mech := kernel.ToMechanism(conn.GetMechanism())

	nsSwitch, err := nsswitch.NewNSSwitch()
	if err != nil {
		return nil, err
	}
	defer func() { _ = nsSwitch.Close() }()

	var clientNetNSHandle netns.NsHandle
	clientNetNSHandle, err = nshandle.FromURL(mech.GetNetNSURL())
	if err != nil {
		return nil, err
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifName := mech.GetInterfaceName(conn)
	err = moveInterfaceToAnotherNamespace(nsSwitch, ifName, clientNetNSHandle, nsSwitch.NetNSHandle)
	if err != nil {
		return nil, err
	}
	logEntry.Infof("moved network interface %s into the Forwarder's namespace for connection %s", ifName, conn.GetId())

	return next.Server(ctx).Close(ctx, conn)
}

func moveInterfaceToAnotherNamespace(nsSwitch *nsswitch.NSSwitch, ifName string, fromNetNS, toNetNS netns.NsHandle) error {
	return nsSwitch.RunIn(fromNetNS, func() error {
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
