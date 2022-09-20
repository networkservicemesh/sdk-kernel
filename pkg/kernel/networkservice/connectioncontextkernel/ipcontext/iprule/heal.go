// Copyright (c) 2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

func recoverTableIDs(ctx context.Context, conn *networkservice.Connection, tableIDs *Map) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		_, ok := tableIDs.Load(conn.GetId())
		if ok {
			return nil
		}

		if len(conn.Context.IpContext.Policies) == 0 {
			return nil
		}

		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer netlinkHandle.Close()

		podRules, err := netlinkHandle.RuleList(netlink.FAMILY_ALL)
		if err != nil {
			return errors.WithStack(err)
		}

		// try to find the corresponding missing policies in the network namespace of the pod
		for _, policy := range conn.Context.IpContext.Policies {
			policyRule, err := policyToRule(policy)
			if err != nil {
				return errors.WithStack(err)
			}
			for i := range podRules {
				if ruleEquals(&podRules[i], policyRule) {
					log.FromContext(ctx).
						WithField("From", policy.From).
						WithField("IPProto", policy.Proto).
						WithField("DstPort", policy.DstPort).
						WithField("SrcPort", policy.SrcPort).
						WithField("Table", podRules[i].Table).Debug("policy recovered")
					err := delRule(ctx, netlinkHandle, policy, podRules[i].Table)
					if err != nil {
						return errors.WithStack(err)
					}
					break
				}
			}
		}
	}
	return nil
}

func ruleEquals(a, b *netlink.Rule) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Src.String() == b.Src.String() && a.IPProto == b.IPProto && rulePortRangeEquals(a.Dport, b.Dport) && rulePortRangeEquals(a.Sport, b.Sport)
}

func rulePortRangeEquals(a, b *netlink.RulePortRange) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Start == b.Start && a.End == b.End
}
