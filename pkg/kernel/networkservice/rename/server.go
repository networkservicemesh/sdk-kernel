// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package rename provides link rename chain element
package rename

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type renameServer struct {
	// If we have 2 rename servers in chain, they should manage their own mappings.
	id string
}

// NewServer returns a new link rename server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &renameServer{
		id: uuid.New().String(),
	}
}

func (s *renameServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mech := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mech == nil {
		return next.Server(ctx).Request(ctx, request)
	}
	ifName := mech.GetInterfaceName()

	vfConfig := vfconfig.Config(ctx)
	if vfConfig == nil || vfConfig.VFInterfaceName == ifName {
		return next.Server(ctx).Request(ctx, request)
	}

	if err := renameLink(vfConfig.VFInterfaceName, ifName); err != nil {
		return nil, err
	}
	oldIfName := vfConfig.VFInterfaceName
	vfConfig.VFInterfaceName = ifName

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if renameErr := renameLink(ifName, oldIfName); renameErr != nil {
			err = errors.Wrapf(err, renameErr.Error())
		}
		return nil, err
	}

	if _, renamed := loadOldIfName(ctx, s.id); !renamed {
		storeOldIfName(ctx, s.id, oldIfName)
	}

	return conn, nil
}

func (s *renameServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)

	var renameErr error
	if mech := kernel.ToMechanism(conn.GetMechanism()); mech != nil {
		ifName := mech.GetInterfaceName()
		if oldIfName, renamed := loadOldIfName(ctx, s.id); renamed {
			renameErr = renameLink(ifName, oldIfName)
		}
	}

	if err != nil && renameErr != nil {
		return nil, errors.Wrap(err, renameErr.Error())
	}
	if renameErr != nil {
		return nil, renameErr
	}
	return &empty.Empty{}, err
}

func renameLink(oldName, newName string) error {
	link, err := netlink.LinkByName(oldName)
	if err != nil {
		return errors.Wrapf(err, "failed to get the net interface: %v", oldName)
	}

	if err = netlink.LinkSetName(link, newName); err != nil {
		return errors.Wrapf(err, "failed to rename net interface: %v -> %v", oldName, newName)
	}

	return nil
}
