// Copyright (c) 2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2021-2022 Nordix Foundation.
//
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

//go:build linux

package iprule

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

func recoverTableIDs(ctx context.Context, conn *networkservice.Connection, tableIDs *genericsync.Map[string, policies], nsRTableNextIDToConnID *genericsync.Map[netnsRTableNextID, string]) error {
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
			return err
		}
		defer netlinkHandle.Close()

		ifName := mechanism.GetInterfaceName()
		l, err := netlinkHandle.LinkByName(ifName)
		if err != nil {
			return errors.Wrapf(err, "failed to find link %s", ifName)
		}

		podRules, err := netlinkHandle.RuleList(netlink.FAMILY_ALL)
		if err != nil {
			return errors.Wrap(err, "failed to get list of rules")
		}

		tableIDtoPolicyMap := make(map[int]*networkservice.PolicyRoute)
		// try to find the corresponding missing policies in the network namespace of the pod
		for _, policy := range conn.Context.IpContext.Policies {
			policyRule, err := policyToRule(policy)
			if err != nil {
				return err
			}
			for i := range podRules {
				if ruleEquals(&podRules[i], policyRule) {
					tableIDtoPolicyMap[podRules[i].Table] = policy
					log.FromContext(ctx).
						WithField("From", policy.From).
						WithField("IPProto", policy.Proto).
						WithField("DstPort", policy.DstPort).
						WithField("SrcPort", policy.SrcPort).
						WithField("Table", podRules[i].Table).Debug("policy recovered")
					break
				}
			}
		}

		return deleteRemainders(ctx, netlinkHandle, tableIDtoPolicyMap, podRules, l, mechanism.GetNetNSURL(), nsRTableNextIDToConnID)
	}
	return nil
}

func deleteRemainders(ctx context.Context, netlinkHandle *netlink.Handle, tableIDtoPolicyMap map[int]*networkservice.PolicyRoute, podRules []netlink.Rule, l netlink.Link, mechanismNetNSURL string, nsRTableNextIDToConnID *genericsync.Map[netnsRTableNextID, string]) error {
	// Get netns for key to namespace to routing tableID map
	netNS, err := nshandle.FromURL(mechanismNetNSURL)
	if err != nil {
		return err
	}
	for tableID, policy := range tableIDtoPolicyMap {
		usage := 0
		for i := range podRules {
			if podRules[i].Table == tableID {
				usage++
			}
		}
		if usage == 1 {
			err := delRule(ctx, netlinkHandle, policy, tableID, l.Attrs().Index, createNetnsRTableNextID(netNS.UniqueId(), tableID), nsRTableNextIDToConnID)
			if err != nil {
				return err
			}
		} else {
			err := delRuleOnly(ctx, netlinkHandle, policy)
			if err != nil {
				return err
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
