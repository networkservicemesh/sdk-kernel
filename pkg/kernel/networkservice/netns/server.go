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

// Package netns provides chain element to switch net NS
package netns

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/nsswitch"
)

type netNSServer struct{}

// NewServer returns a new net NS server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &netNSServer{}
}

func (s *netNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	rv, err := runInNetNS(request.GetConnection().GetMechanism(), func() (interface{}, error) {
		return next.Server(ctx).Request(ctx, request)
	})
	if err != nil {
		return nil, err
	}
	return rv.(*networkservice.Connection), nil
}

func (s *netNSServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := runInNetNS(conn.GetMechanism(), func() (interface{}, error) {
		return next.Server(ctx).Close(ctx, conn)
	})
	if err != nil {
		return nil, err
	}
	return rv.(*empty.Empty), nil
}

func runInNetNS(mechanism *networkservice.Mechanism, runner func() (interface{}, error)) (interface{}, error) {
	if mech := kernel.ToMechanism(mechanism); mech != nil {
		nsSwitch, err := nsswitch.NewNSSwitch()
		if err != nil {
			return nil, errors.Wrap(err, "failed to init NS switch")
		}
		defer func() { _ = nsSwitch.Close() }()

		clientNetNSHandle, err := netns.GetFromPath(mech.GetNetNSURL())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get client net NS: %v", mech.GetNetNSURL())
		}
		defer func() { _ = clientNetNSHandle.Close() }()

		if err = nsSwitch.SwitchTo(clientNetNSHandle); err != nil {
			return nil, errors.Wrapf(err, "failed to switch to the client net NS: %v", mech.GetNetNSURL())
		}
		defer func() {
			if err = nsSwitch.SwitchBack(); err != nil {
				panic(errors.Wrapf(err, "failed to switch to the forwarder net NS: %v", nsSwitch.NetNSHandle))
			}
		}()
	}
	return runner()
}
