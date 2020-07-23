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
	"runtime"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/utils"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	sdkKernel "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
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

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}
	defer func() { _ = forwarderNetNSHandle.Close() }()

	clientNetNSHandle, err := utils.GetNSHandleFromInode(mech.GetNetNSInode())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifaceName := mech.GetParameters()[kernel.InterfaceNameKey]
	if ifaceName == "" {
		return nil, errors.New("Interface name is not found")
	}

	err = a.moveInterfaceToAnotherNamespace(ifaceName, forwarderNetNSHandle, clientNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to move network interface %s into the Client's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Client's namespace for connection %s", ifaceName, connID)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		errMovingBack := a.moveInterfaceToAnotherNamespace(ifaceName, clientNetNSHandle, forwarderNetNSHandle)
		if errMovingBack != nil {
			log.Entry(ctx).Infof("Failed to move network interface %s into the Forwarder's namespace for connection %s", ifaceName, connID)
		}
		log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, connID)
	}
	return conn, err
}

func (a *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	mech := kernel.ToMechanism(conn.GetMechanism())

	/* Lock the OS thread so we don't accidentally switch namespaces */
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}
	defer func() { _ = forwarderNetNSHandle.Close() }()

	clientNetNSHandle, err := utils.GetNSHandleFromInode(mech.GetNetNSInode())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}
	defer func() { _ = clientNetNSHandle.Close() }()

	ifaceName := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if ifaceName == "" {
		return nil, errors.New("Interface name is not found")
	}

	err = a.moveInterfaceToAnotherNamespace(ifaceName, clientNetNSHandle, forwarderNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Forwarder's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, conn.GetId())

	return next.Server(ctx).Close(ctx, conn)
}

func (a *injectServer) moveInterfaceToAnotherNamespace(ifaceName string, fromNetNS, toNetNS netns.NsHandle) error {
	link, err := sdkKernel.FindHostDevice("", ifaceName, fromNetNS)
	if err != nil {
		return err
	}

	err = link.MoveToNetns(toNetNS)
	if err != nil {
		return errors.Wrapf(err, "Failed to move interface %s to another namespace", ifaceName)
	}

	return nil
}
