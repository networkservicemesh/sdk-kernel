// Copyright (c) 2022 Xored Software Inc and others.
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

package iptables4nattemplate

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

type iptablesClient struct {
	manager IPTablesManager
}

// NewClient - returns a new networkservice.NetworkServiceClient that modify IPTables rules
// by mechanism provided template on Request and rollbacks rules changes on Close
func NewClient() networkservice.NetworkServiceClient {
	return &iptablesClient{
		manager: &iptableManagerImpl{},
	}
}

func (c *iptablesClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err := applyIptablesRules(ctx, conn, c); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := c.Close(closeCtx, conn, opts...); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (c *iptablesClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	var restoreErr error
	ctxMap := metadata.Map(ctx, metadata.IsClient(c))
	if initialRules, rulesWasApplied := ctxMap.Load(applyIPTablesKey{}); rulesWasApplied {
		mechanism := kernel.ToMechanism(conn.GetMechanism())
		currentNsHandler, handleErr := nshandle.Current()
		if handleErr != nil {
			return nil, handleErr
		}
		defer func() { _ = currentNsHandler.Close() }()

		targetHsHandler, handleErr := nshandle.FromURL(mechanism.GetNetNSURL())
		if handleErr != nil {
			return nil, handleErr
		}
		defer func() { _ = targetHsHandler.Close() }()

		restoreErr = nshandle.RunIn(currentNsHandler, targetHsHandler, func() error {
			return c.manager.Restore(initialRules.(string))
		})
	}

	if err != nil && restoreErr != nil {
		return nil, errors.Wrap(err, restoreErr.Error())
	}
	if restoreErr != nil {
		return nil, restoreErr
	}

	return &empty.Empty{}, err
}
