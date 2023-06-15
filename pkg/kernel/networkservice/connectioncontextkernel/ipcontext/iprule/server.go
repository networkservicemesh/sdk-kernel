// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Nordix Foundation.
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

package iprule

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
	"github.com/pkg/errors"
)

type ipruleServer struct {
	tables Map
	// Protecting route and rule setting with this sync.Map
	// The next table ID is calculated based on a dump
	// other connection from same client can add new table in parallel
	nsRTableNextIDToConnID NetnsRTableNextIDToConnMap
}

// NewServer creates a new server chain element setting ip rules
func NewServer() networkservice.NetworkServiceServer {
	return &ipruleServer{}
}

func (i *ipruleServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	err = recoverTableIDs(ctx, conn, &i.tables)
	if err != nil {
		return nil, err
	}

	if err := create(ctx, conn, &i.tables, &i.nsRTableNextIDToConnID); err != nil {
		closeCtx, cancelClose := postponeCtxFunc()
		defer cancelClose()

		if _, closeErr := i.Close(closeCtx, conn); closeErr != nil {
			err = errors.Wrapf(err, "connection closed with error: %s", closeErr.Error())
		}

		return nil, err
	}

	return conn, nil
}

func (i *ipruleServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_ = del(ctx, conn, &i.tables, &i.nsRTableNextIDToConnID)
	return next.Server(ctx).Close(ctx, conn)
}
