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

//+build !windows

// Package xconnectns provides an Endpoint implementing the kernel Forwarder networks service
package xconnectns

import (
	"context"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/netnsconnectioncontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type kernelServer struct {
	endpoint.Endpoint
}

// NewServer - returns an Endpoint implementing the Kernel Forwarder networks service
//             - name - name of the Forwarder
//             - authzServer - policy for allowing or rejecting requests
//             - tokenGenerator - token.GeneratorFunc - generates tokens for use in Path
//             - clientUrl - *url.URL for the talking to the NSMgr
//             - ...clientDialOptions - dialOptions for dialing the NSMgr
func NewServer(
	ctx context.Context,
	name string,
	authzServer networkservice.NetworkServiceServer,
	tokenGenerator token.GeneratorFunc,
	clientURL *url.URL,
	clientDialOptions ...grpc.DialOption,
) endpoint.Endpoint {
	rv := kernelServer{}

	rv.Endpoint = endpoint.NewServer(ctx,
		tokenGenerator,
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(authzServer),
		endpoint.WithAdditionalFunctionality(
			recvfd.NewServer(),
			clienturl.NewServer(clientURL),
			heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
			connect.NewServer(ctx,
				client.NewCrossConnectClientFactory(
					client.WithName(name),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						// mechanism
						vlan.NewClient(),
						recvfd.NewClient(),
						sendfd.NewClient(),
					),
				),
				connect.WithDialOptions(clientDialOptions...),
			),
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				vlanmech.MECHANISM: vlan.NewServer(),
			}),
			// setup IP and route context
			netnsconnectioncontext.NewServer(),
			sendfd.NewServer(),
		),
	)

	return rv
}
