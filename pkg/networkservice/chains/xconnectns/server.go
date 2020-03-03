// Copyright (c) 2020 Intel Corporation. All Rights Reserved.
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

package xconnectns

import (
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"

	"github.com/networkservicemesh/sdk-kernel/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-kernel/pkg/networkservice/connectioncontext/ipcontext/routes"
	"github.com/networkservicemesh/sdk-kernel/pkg/networkservice/mechanisms/kernel"
)

type xconnectNSServer struct {
	endpoint.Endpoint
}

// NewServer - returns a new Endpoint implementing the XConnect Network Service for use as a Forwarder
//             name - name of the Forwarder
//             u - *url.URL for the talking to the NSMgr
func NewServer(name string, u *url.URL) endpoint.Endpoint {
	server := xconnectNSServer{}
	server.Endpoint = endpoint.NewServer(
		name,
		// Preference ordered list of mechanisms we support for incoming connections
		kernel.NewServer(),
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(u),
		connect.NewServer(
			client.NewClientFactory(
				name,
				// What to call onHeal
				addressof.NetworkServiceClient(adapters.NewServerToClient(server)),
				// Preference ordered list of mechanisms we support for outgoing connections
				kernel.NewClient()),
		),
		ipaddress.NewServer(),
		routes.NewServer(),
	)
	return server
}
