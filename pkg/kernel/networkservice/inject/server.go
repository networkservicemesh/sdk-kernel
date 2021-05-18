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

// Package inject contains chain element that moves network interface to and from a Client's pod network namespace
package inject

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type injectServer struct{}

// NewServer - returns a new networkservice.NetworkServiceServer that moves given network interface into the Client's
// pod network namespace on Request and back to Forwarder's network namespace on Close
func NewServer() networkservice.NetworkServiceServer {
	return &injectServer{}
}

func (s *injectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("injectServer", "Request")

	mech := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mech == nil {
		return next.Server(ctx).Request(ctx, request)
	}

	if err := move(logger, request.GetConnection(), false); err != nil {
		return nil, err
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		err = move(logger, request.GetConnection(), true)
		if err != nil {
			logger.Warnf("server request failed, failed to move back the interface: %v", err)
		}
	}
	return conn, err
}

func (s *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("injectServer", "Close")

	_, err := next.Server(ctx).Close(ctx, conn)

	injectErr := move(logger, conn, true)

	if err != nil && injectErr != nil {
		return nil, errors.Wrap(err, injectErr.Error())
	}
	if injectErr != nil {
		return nil, injectErr
	}
	return &empty.Empty{}, err
}
