// Copyright (c) 2022 Xored Software Inc and others.
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

// Package routelocalnet provides chain element that enables route_localnet flat for connection network interface
package routelocalnet

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type routeLocalNetServer struct {
}

// NewServer - returns a new networkservice.NetworkServiceServer that writes route_localnet flag
// for network interface on Request if enabled in mechanism
func NewServer() networkservice.NetworkServiceServer {
	return &routeLocalNetServer{}
}

func (s *routeLocalNetServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	mechanism := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mechanism != nil && mechanism.GetRouteLocalNet() {
		fo, err := os.Create(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/route_localnet", mechanism.GetInterfaceName()))
		if err != nil {
			return nil, err
		}

		defer func() { _ = fo.Close() }()

		_, err = fo.WriteString("1")
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func (s *routeLocalNetServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
