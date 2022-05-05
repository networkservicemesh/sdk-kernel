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
	"sync/atomic"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/tatsushid/go-fastping"
)

const (
	defaultTimeout = 200 * time.Millisecond
)

// NewKernelLivenessCheck is an implementation of heal.LivenessCheck. It sends ICMP
// ping and checks reply. Returns false if didn't get reply.
func NewKernelLivenessCheck(deadlineCtx context.Context, conn *networkservice.Connection) bool {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism == nil {
		log.FromContext(deadlineCtx).Errorf("Ping failed: wrong mechanism type")
	}

	p := fastping.NewPinger()
	deadline, ok := deadlineCtx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultTimeout)
	}

	p.MaxRTT = time.Until(deadline)

	addrCount := len(conn.GetContext().GetIpContext().GetDstIpAddrs())
	for _, cidr := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
		addr, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return false
		}
		ipAddr := &net.IPAddr{IP: addr}
		p.AddIPAddr(ipAddr)
	}

	var count int32 = 0

	p.OnRecv = func(ipAddr *net.IPAddr, d time.Duration) {
		atomic.AddInt32(&count, 1)
	}

	var aliveCh = make(chan bool)
	p.OnIdle = func() {
		aliveCh <- int(atomic.LoadInt32(&count)) == addrCount
		close(aliveCh)
	}

	err := p.Run()

	if err != nil {
		log.FromContext(deadlineCtx).Error("Ping failed: %s", err.Error())
		return false
	}

	select {
	case alive := <-aliveCh:
		return alive
	case <-deadlineCtx.Done():
		return false
	}
}
