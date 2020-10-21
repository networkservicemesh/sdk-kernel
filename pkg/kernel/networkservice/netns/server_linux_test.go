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

package netns_test

import (
	"context"
	"net/url"
	"path"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	netnschain "github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/netns"
)

const (
	netNSPath   = "/run/netns"
	equalFormat = "Not equal: \n" +
		"expected: %v\n" +
		"actual  : %v"
)

func TestNetNSServer(t *testing.T) {
	netNSName := uuid.New().String()
	newHandle, err := func() (netns.NsHandle, error) {
		baseHandle, err := netns.Get()
		require.NoError(t, err)

		defer func() {
			_ = netns.Set(baseHandle)
			_ = baseHandle.Close()
		}()
		return netns.NewNamed(netNSName)
	}()
	require.NoError(t, err)
	defer func() {
		_ = newHandle.Close()
		_ = netns.DeleteNamed(netNSName)
	}()

	server := chain.NewNetworkServiceServer(
		netnschain.NewServer(),
		&checkNetNSServer{
			t:      t,
			handle: newHandle,
		},
	)

	conn, err := server.Request(context.TODO(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Mechanism: &networkservice.Mechanism{
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					kernel.NetNSURL: (&url.URL{Scheme: "file", Path: path.Join(netNSPath, netNSName)}).String(),
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = server.Close(context.TODO(), conn)
	require.NoError(t, err)
}

type checkNetNSServer struct {
	t      *testing.T
	handle netns.NsHandle
}

func (s *checkNetNSServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	handle, err := netns.Get()
	require.NoError(s.t, err)
	defer func() { _ = handle.Close() }()

	require.True(s.t, s.handle.Equal(handle), equalFormat, s.handle, handle)

	return next.Server(ctx).Request(ctx, request)
}

func (s *checkNetNSServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	handle, err := netns.Get()
	require.NoError(s.t, err)
	defer func() { _ = handle.Close() }()

	require.True(s.t, s.handle.Equal(handle), equalFormat, s.handle, handle)

	return next.Server(ctx).Close(ctx, conn)
}
