// Copyright (C) 2021, Nordix Foundation
//
// All rights reserved. This program and the accompanying materials
// are made available under the terms of the Apache License, Version 2.0
// which accompanies this distribution, and is available at
// http://www.apache.org/licenses/LICENSE-2.0

package inject

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type injectClient struct{}

// NewClient - returns a new networkservice.NetworkServiceClient that moves given network
// interface into the Endpoint's pod network namespace on Request and back to Forwarder's
// network namespace on Close
func NewClient() networkservice.NetworkServiceClient {
	return &injectClient{}
}

func (c *injectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest,
	opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("injectClient", "Request")
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if err := move(logger, conn, false); err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *injectClient) Close(ctx context.Context, conn *networkservice.Connection,
	opts ...grpc.CallOption) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("injectClient", "Close")

	rv, err := next.Client(ctx).Close(ctx, conn, opts...)

	injectErr := move(logger, conn, true)

	if err != nil && injectErr != nil {
		return nil, errors.Wrap(err, injectErr.Error())
	}
	if injectErr != nil {
		return nil, injectErr
	}
	return rv, err
}
