// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021 Nordix Foundation.
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

// Package ethernetcontext provides chain element for setup link ethernet properties
package ethernetcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type vfEthernetContextServer struct{}

// NewVFServer returns a new VF ethernet context server chain element
func NewVFServer() networkservice.NetworkServiceServer {
	return &vfEthernetContextServer{}
}

func (s *vfEthernetContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if vfConfig, ok := vfconfig.Load(ctx, false); ok {
		err := vfCreate(vfConfig, request.Connection, false)
		if err != nil {
			return nil, err
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (s *vfEthernetContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
