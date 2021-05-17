// Copyright (c) 2020 Nordix Foundation.
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

package rename

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type renameClient struct {
}

// NewClient returns a new link rename client chain element, This client is mostly useful for
// forwarder's which connects pod container with sriov vf device using VFConfig.
func NewClient() networkservice.NetworkServiceClient {
	return &renameClient{}
}

func (c *renameClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest,
	opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("renameClient", "Request")
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	mech := kernel.ToMechanism(conn.GetMechanism())
	if mech == nil {
		return conn, nil
	}

	ifName := mech.GetInterfaceName()

	vfConfig := vfconfig.Config(ctx)
	if vfConfig == nil || vfConfig.VFInterfaceName == ifName {
		return conn, nil
	}

	if err := renameLink(vfConfig.VFInterfaceName, ifName); err != nil {
		return nil, err
	}
	logger.Infof("renamed the interface %s into %s", vfConfig.VFInterfaceName, ifName)
	return conn, nil
}

func (c *renameClient) Close(ctx context.Context, conn *networkservice.Connection,
	opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("renameClient", "Close")

	rv, err := next.Client(ctx).Close(ctx, conn, opts...)

	var renameErr error
	if mech := kernel.ToMechanism(conn.GetMechanism()); mech != nil {
<<<<<<< HEAD
		ifName := mech.GetInterfaceName()
		_, err := netlink.LinkByName(ifName)
=======
		ifName := mech.GetInterfaceName(conn)
		_, err = netlink.LinkByName(ifName)
>>>>>>> 1a29404 (fix lint issues)
		if err == nil {
			vfConfig := vfconfig.Config(ctx)
			renameErr = renameLink(ifName, vfConfig.VFInterfaceName)
			logger.Infof("renamed interface %s back into original name %s", ifName, vfConfig.VFInterfaceName)
		}
	}

	if err != nil && renameErr != nil {
		return nil, errors.Wrap(err, renameErr.Error())
	}
	if renameErr != nil {
		return nil, renameErr
	}
	return rv, nil
}
