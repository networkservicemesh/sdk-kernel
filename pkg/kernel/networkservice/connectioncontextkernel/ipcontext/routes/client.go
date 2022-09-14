// Copyright (c) 2020-2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package routes

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type routesClient struct{}

// NewClient creates a NetworkServiceClient that will put the routes from the connection context into
//
//	the kernel network namespace kernel interface being inserted iff the
//	selected mechanism for the connection is a kernel mechanism
//	           Client
//	+- - - - - - - - - - - - - - - -+         +---------------------------+
//	|                               |         |  kernel network namespace |
//	                                          |                           |
//	|                               |         |                           |
//	                                          |                           |
//	|                               |         |                           |
//	                                          |                           |
//	|                               |         |                           |
//	                                +--------- ---------+                 |
//	|                               |         |                           |
//	                                          |                           |
//	|                               |         |                           |
//	                                          |      routes.Client()      |
//	|                               |         |                           |
//	                                          |                           |
//	|                               |         |                           |
//	+- - - - - - - - - - - - - - - -+         +---------------------------+
func NewClient() networkservice.NetworkServiceClient {
	return &routesClient{}
}

func (i *routesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := create(ctx, conn, metadata.IsClient(i)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := i.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (i *routesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	// We do not have to delete routes here because the kernel deletes routes for us when we delete the interface
	return next.Client(ctx).Close(ctx, conn, opts...)
}
