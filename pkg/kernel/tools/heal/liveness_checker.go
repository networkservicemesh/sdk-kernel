// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package heal contains an implementation of LivenessChecker.
package heal

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/tatsushid/go-fastping"
)

var _ heal.LivenessChecker = ICMPLivenessChecker

// ICMPLivenessChecker is an implementation of LivenessChecker. It sends ICMP
// pings continuously and waits for replies until the first missing reply or
// timeout.
func ICMPLivenessChecker(deadlineCtx context.Context, conn *networkservice.Connection) bool {
	deadline, _ := deadlineCtx.Deadline()
	p := fastping.NewPinger()
	p.MaxRTT = time.Until(deadline)

	addrs := make(map[string]int)
	for _, cidr := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
		addr, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return false
		}
		ipAddr := &net.IPAddr{IP: addr}
		addrs[ipAddr.String()] = 0
		p.AddIPAddr(ipAddr)
	}

	var mu sync.Mutex

	p.OnRecv = func(ipAddr *net.IPAddr, d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		addrs[ipAddr.String()]++
	}

	alive := true
	p.OnIdle = func() {
		mu.Lock()
		defer mu.Unlock()
		for _, count := range addrs {
			if count == 0 {
				alive = false
				return
			}
		}
	}

	err := p.Run()

	if err != nil {
		return false
	}

	return alive
}
