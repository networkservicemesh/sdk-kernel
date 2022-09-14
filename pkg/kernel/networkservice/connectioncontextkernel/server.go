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

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/ipneighbors"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/iprule"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/ipcontext/routes"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontextkernel/mtu"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
)

// NewServer provides a NetworkServiceServer that applies the connection context to a kernel interface
// It applies the connection context on the *kernel* side of an interface plugged into the
// Endpoint.  Generally only used by privileged Endpoints like those implementing
// the Cross Connect Network Service for K8s (formerly known as NSM Forwarder).
//
//	                                                    Endpoint
//	                                          +---------------------------+
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                      +-------------------+                           |
//	connectioncontextkernel.NewServer()       |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          |                           |
//	                                          +---------------------------+
func NewServer() networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		mtu.NewServer(),
		ipneighbors.NewServer(),
		iprule.NewServer(),
		routes.NewServer(),
		ipaddress.NewServer(),
	)
}
