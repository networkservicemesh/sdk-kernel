// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
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

package connectioncontextkernel

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/iptables4nattemplate"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/mtu"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/routelocalnet"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/ipneighbors"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/routes"
)

// NewClient provides a NetworkServiceClient that applies the connectioncontext to a kernel interface
// It applies the connectioncontext on the *kernel* side of an interface leaving the
// Client.  Generally only used by privileged Clients like those implementing
// the Cross Connect Network Service for K8s (formerly known as NSM Forwarder).
//
//	           Client
//	+---------------------------+
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           +-------------------+
//	|                           |          connectioncontextkernel.NewClient()
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	|                           |
//	+---------------------------+
func NewClient() networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		mtu.NewClient(),
		ipneighbors.NewClient(),
		routes.NewClient(),
		ipaddress.NewClient(),
		routelocalnet.NewClient(),
		iptables4nattemplate.NewClient(),
	)
}
