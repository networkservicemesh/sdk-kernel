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

package vlan

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"google.golang.org/grpc"
)

type vlanClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &vlanClient{}
}

func (k *vlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	getMechPref := func(class string) networkservice.Mechanism {
		return networkservice.Mechanism{
			Cls:        class,
			Type:       vlanmech.MECHANISM,
			Parameters: make(map[string]string),
		}
	}
	localMechanism := getMechPref(cls.LOCAL)
	request.MechanismPreferences = append(request.MechanismPreferences, &localMechanism)
	remoteMechanism := getMechPref(cls.REMOTE)
	request.MechanismPreferences = append(request.MechanismPreferences, &remoteMechanism)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	/* if err := create(ctx, conn, metadata.IsClient(k)); err != nil {
		_, _ = k.Close(ctx, conn, opts...)
		return nil, err
	} */
	return conn, nil
}

func (k *vlanClient) Close(
	ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
