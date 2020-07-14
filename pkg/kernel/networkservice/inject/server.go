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
	if request.GetConnection().GetMechanism().GetType() != kernel.MECHANISM {
		return next.Server(ctx).Request(ctx, request)
	}

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}
	clientNetNSHandle, err := a.getClientNetNSHandle(request.GetConnection())
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}
	ifaceName := request.GetConnection().GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if ifaceName == "" {
		return nil, errors.New("Virtual function's interface name is not found")
	}

	err = a.moveInterfaceToAnotherNamespace(ifaceName, forwarderNetNSHandle, clientNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Client's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Client's namespace for connection %s", ifaceName, request.GetConnection().GetId())

	return next.Server(ctx).Request(ctx, request)
}

func (a *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if conn.GetMechanism().GetType() != kernel.MECHANISM {
		return next.Server(ctx).Close(ctx, conn)
	}

	forwarderNetNSHandle, err := netns.Get()
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Forwarder's network namespace handle")
	}

	clientNetNSHandle, err := a.getClientNetNSHandle(conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to obtain Client's network namespace handle")
	}

	ifaceName := conn.GetMechanism().GetParameters()[kernel.InterfaceNameKey]
	if ifaceName == "" {
		return nil, errors.New("Virtual function's interface name is not found")
	}

	err = a.moveInterfaceToAnotherNamespace(ifaceName, clientNetNSHandle, forwarderNetNSHandle)
	if err != nil {
		return nil, errors.Wrapf(err, "Unable to move network interface %s into the Forwarder's namespace", ifaceName)
	}
	log.Entry(ctx).Infof("Moved network interface %s into the Forwarder's namespace for connection %s", ifaceName, conn.GetId())

	return next.Client(ctx).Close(ctx, conn)
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

func (a *injectServer) getClientNetNSHandle(conn *networkservice.Connection) (netns.NsHandle, error) {
	clientNetNSInode := conn.GetMechanism().GetParameters()[kernel.NetNSInodeKey]
	if clientNetNSInode == "" {
		return 0, errors.New("Client's pod net ns inode is not found")
	}

	return utils.GetNSHandleFromInode(clientNetNSInode)
}
