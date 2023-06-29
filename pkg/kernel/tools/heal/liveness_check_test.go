// Copyright (c) 2023 Cisco and/or its affiliates.
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

package heal_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/heal"
)

const unPingableIPv4 = "172.168.1.1"
const unPingableIPv6 = "2005::1"

func createConnection(srcIPs, dstIPs []string) *networkservice.Connection {
	return &networkservice.Connection{
		Mechanism: &networkservice.Mechanism{
			Type: kernel.MECHANISM,
		},
		Context: &networkservice.ConnectionContext{IpContext: &networkservice.IPContext{
			SrcIpAddrs: srcIPs,
			DstIpAddrs: dstIPs,
		}},
	}
}
func Test_LivenessChecker(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	samples := []struct {
		Name           string
		Connection     *networkservice.Connection
		PingersCount   int32
		ExpectedResult bool
	}{
		{
			Name: "Pingable IPv4 one pair",
			Connection: createConnection(
				[]string{"172.168.0.1/32"},
				[]string{"172.168.0.2/32"},
			),
			PingersCount:   1,
			ExpectedResult: true,
		},
		{
			Name: "Pingable IPv4 two pairs",
			Connection: createConnection(
				[]string{"172.168.0.1/32", "172.168.0.3/32"},
				[]string{"172.168.0.2/32", "172.168.0.4/32"},
			),
			PingersCount:   4,
			ExpectedResult: true,
		},
		{
			Name: "Unpingable IPv4 two pairs",
			Connection: createConnection(
				[]string{"172.168.0.1/32", "172.168.0.3/32"},
				[]string{"172.168.0.2/32", unPingableIPv4 + "/32"},
			),
			PingersCount:   4,
			ExpectedResult: false,
		},
		{
			Name: "Pingable IPv4 and IPv6",
			Connection: createConnection(
				[]string{"172.168.0.1/32", "2004::1/128"},
				[]string{"172.168.0.2/32", "2004::2/128"},
			),
			PingersCount:   2,
			ExpectedResult: true,
		},
		{
			Name: "Unpingable IPv4 and IPv6",
			Connection: createConnection(
				[]string{"172.168.0.1/32", "2004::1/128"},
				[]string{"172.168.0.2/32", unPingableIPv6 + "/128"},
			),
			PingersCount:   2,
			ExpectedResult: false,
		},
		{
			Name: "SrcIPs is empty",
			Connection: createConnection(
				[]string{},
				[]string{"172.168.0.2/32"},
			),
			PingersCount:   0,
			ExpectedResult: true,
		},
		{
			Name: "DstIPs is empty",
			Connection: createConnection(
				[]string{"172.168.0.1/32"},
				[]string{},
			),
			PingersCount:   0,
			ExpectedResult: true,
		},
	}
	for _, s := range samples {
		sample := s
		t.Run(sample.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			pingerFactory := &testPingerFactory{}
			ok := heal.KernelLivenessCheckWithOptions(ctx, sample.Connection, heal.WithPingerFactory(pingerFactory))
			require.Equal(t, sample.ExpectedResult, ok)
			require.Equal(t, pingerFactory.pingersCount, sample.PingersCount)
		})
	}
}

type testPingerFactory struct {
	pingersCount int32
}

func (p *testPingerFactory) CreatePinger(_, dstIP string, _ time.Duration, count int) heal.Pinger {
	atomic.AddInt32(&p.pingersCount, 1)
	return &testPinger{
		dstIP: dstIP,
		count: count,
	}
}

type testPinger struct {
	dstIP string
	count int
}

func (p *testPinger) Run() error {
	return nil
}

func (p *testPinger) GetReceivedPackets() int {
	if p.dstIP == unPingableIPv4 || p.dstIP == unPingableIPv6 {
		return 0
	}
	return p.count
}
