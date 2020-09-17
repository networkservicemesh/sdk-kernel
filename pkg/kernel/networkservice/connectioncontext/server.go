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

// Package connectioncontext provides chain element for setup link connection properties
package connectioncontext

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontext/ethernetcontext"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/connectioncontext/ipcontext"
)

// NewServer returns connection context server chain element
func NewServer() networkservice.NetworkServiceServer {
	return chain.NewNetworkServiceServer(
		ethernetcontext.NewServer(),
		ipcontext.NewServer(),
	)
}
