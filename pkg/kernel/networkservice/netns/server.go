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
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nsswitch"
)

type netNSServer struct{}

// NewServer returns a new net NS server chain element
func NewServer() networkservice.NetworkServiceServer {
	return &netNSServer{}
}

func (s *netNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	if mech := kernel.ToMechanism(request.GetConnection().GetMechanism()); mech != nil {
		var nsSwitch *nsswitch.NSSwitch
		var clientNetNSHandle netns.NsHandle
		nsSwitch, clientNetNSHandle, err = nsswitch.NewNSSwitchAndHandle(mech.GetNetNSURL())
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = nsSwitch.Close()
			_ = clientNetNSHandle.Close()
		}()

		err = nsSwitch.RunIn(clientNetNSHandle, func() error {
			conn, err = next.Server(ctx).Request(ctx, request)
			return err
		})
	} else {
		conn, err = next.Server(ctx).Request(ctx, request)
	}
	return conn, err
}

func (s *netNSServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	if mech := kernel.ToMechanism(conn.GetMechanism()); mech != nil {
		var nsSwitch *nsswitch.NSSwitch
		var clientNetNSHandle netns.NsHandle
		nsSwitch, clientNetNSHandle, err = nsswitch.NewNSSwitchAndHandle(mech.GetNetNSURL())
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = nsSwitch.Close()
			_ = clientNetNSHandle.Close()
		}()

		err = nsSwitch.RunIn(clientNetNSHandle, func() error {
			_, err = next.Server(ctx).Close(ctx, conn)
			return err
		})
	} else {
		_, err = next.Server(ctx).Close(ctx, conn)
	}
	return &empty.Empty{}, err
}
