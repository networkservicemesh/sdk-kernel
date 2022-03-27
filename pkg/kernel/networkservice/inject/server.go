// Copyright (c) 2022 Cisco and/or its affiliates.
//
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

//go:build linux
// +build linux

// Package inject contains chain element that moves network interface to and from a Client's pod network namespace
package inject

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type injectServer struct {
	vfRefCountMap   map[string]int
	vfRefCountMutex sync.Mutex
}

// NewServer - returns a new networkservice.NetworkServiceServer that moves given network interface into the Client's
// pod network namespace on Request and back to Forwarder's network namespace on Close
func NewServer() networkservice.NetworkServiceServer {
	return &injectServer{vfRefCountMap: make(map[string]int)}
}

func (s *injectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mech := kernel.ToMechanism(request.GetConnection().GetMechanism())
	if mech == nil {
		return next.Server(ctx).Request(ctx, request)
	}

	var isEstablished bool
	if vfConfig, ok := vfconfig.Load(ctx, metadata.IsClient(s)); ok {
		isEstablished = int(vfConfig.ContNetNS) != 0
	}

	if !isEstablished {
		if err := move(ctx, request.GetConnection(), s.vfRefCountMap, &s.vfRefCountMutex, metadata.IsClient(s), false); err != nil {
			return nil, err
		}
	}

	postponeCtxFunc := postpone.ContextWithValues(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !isEstablished {
		moveCtx, cancelMove := postponeCtxFunc()
		defer cancelMove()

		if moveRenameErr := move(moveCtx, request.GetConnection(), s.vfRefCountMap, &s.vfRefCountMutex, metadata.IsClient(s), true); moveRenameErr != nil {
			err = errors.Wrapf(err, "server request failed, failed to move back the interface: %s", moveRenameErr.Error())
		}
	}

	return conn, err
}

func (s *injectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, err := next.Server(ctx).Close(ctx, conn)

	moveRenameErr := move(ctx, conn, s.vfRefCountMap, &s.vfRefCountMutex, metadata.IsClient(s), true)

	if err != nil && moveRenameErr != nil {
		return nil, errors.Wrap(err, moveRenameErr.Error())
	}
	if moveRenameErr != nil {
		return nil, moveRenameErr
	}
	return &empty.Empty{}, err
}
