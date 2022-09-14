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

package ipaddress

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type ipaddressServer struct {
}

// NewServer provides a NetworkServiceServer that sets the IP on a kernel interface
// It sets the IP Address on the *kernel* side of an interface plugged into the
// Endpoint.  Generally only used by privileged Endpoints like those implementing
// the Cross Connect Network Service for K8s (formerly known as NSM Forwarder).
//
//	                                       Endpoint
//	                            +---------------------------+
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	        +-------------------+                           |
//	ipaddress.NewServer()       |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            |                           |
//	                            +---------------------------+
func NewServer() networkservice.NetworkServiceServer {
	return &ipaddressServer{}
}

func (i *ipaddressServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	if err := create(ctx, conn, metadata.IsClient(i)); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := i.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (i *ipaddressServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
