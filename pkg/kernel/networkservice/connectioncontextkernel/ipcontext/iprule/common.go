// Copyright (c) 2022 Doc.ai and/or its affiliates.
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
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sys/unix"

	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

func create(ctx context.Context, conn *networkservice.Connection, tableIDs *Map) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		// Construct the netlink handle for the target namespace for this kernel interface
		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer netlinkHandle.Close()

		l, err := netlinkHandle.LinkByName(mechanism.GetInterfaceName())
		if err != nil {
			return errors.WithStack(err)
		}

		ps, ok := tableIDs.Load(conn.GetId())
		if !ok {
			if len(conn.Context.IpContext.Policies) == 0 {
				return nil
			}
			ps = make(map[int]*networkservice.PolicyRoute)
			tableIDs.Store(conn.GetId(), ps)
		}
		// Get policies to add and to remove
		toAdd, toRemove := getPolicyDifferences(ps, conn.Context.IpContext.Policies)

		// Remove no longer existing policies
		for tableID, policy := range toRemove {
			if err := delRule(ctx, netlinkHandle, policy, tableID); err != nil {
				return err
			}
			delete(ps, tableID)
			tableIDs.Store(conn.GetId(), ps)
		}
		// Add new policies
		for _, policy := range toAdd {
			tableID, err := getFreeTableID(ctx, netlinkHandle)
			if err != nil {
				return errors.Wrapf(err, "failed to get free tableId")
			}
			// If policy doesn't contain any route - add default
			if len(policy.Routes) == 0 {
				policy.Routes = append(policy.Routes, defaultRoute())
			}

			for _, route := range policy.Routes {
				if err := routeAdd(ctx, netlinkHandle, l, route, tableID); err != nil {
					return errors.Wrapf(err, "failed to add route")
				}
			}
			if err := ruleAdd(ctx, netlinkHandle, policy, tableID); err != nil {
				return errors.Wrapf(err, "failed to add rule")
			}
			ps[tableID] = policy
			tableIDs.Store(conn.GetId(), ps)
		}
	}
	return nil
}

func getPolicyDifferences(current map[int]*networkservice.PolicyRoute, newPolicies []*networkservice.PolicyRoute) (toAdd []*networkservice.PolicyRoute, toRemove map[int]*networkservice.PolicyRoute) {
	type table struct {
		tableID     int
		policyRoute *networkservice.PolicyRoute
	}
	toRemove = make(map[int]*networkservice.PolicyRoute)
	currentMap := make(map[string]*table)
	for tableID, policy := range current {
		currentMap[policyKey(policy)] = &table{
			tableID:     tableID,
			policyRoute: policy,
		}
	}
	for _, policy := range newPolicies {
		if _, ok := currentMap[policyKey(policy)]; !ok {
			toAdd = append(toAdd, policy)
		}
		delete(currentMap, policyKey(policy))
	}
	for _, table := range currentMap {
		toRemove[table.tableID] = table.policyRoute
	}
	return toAdd, toRemove
}

func policyKey(policy *networkservice.PolicyRoute) string {
	return fmt.Sprintf("%s;%s;%s;%s", policy.DstPort, policy.SrcPort, policy.From, policy.Proto)
}

func policyToRule(policy *networkservice.PolicyRoute) (*netlink.Rule, error) {
	rule := netlink.NewRule()
	if policy.From != "" {
		src, err := netlink.ParseIPNet(policy.From)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		rule.Src = src
	}
	if policy.Proto != "" {
		protocol, err := strconv.Atoi(policy.Proto)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		rule.IPProto = protocol
	}
	dstPortRange, err := networkservice.ParsePortRange(policy.DstPort)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if dstPortRange != nil {
		rule.Dport = netlink.NewRulePortRange(dstPortRange.Start, dstPortRange.End)
	}
	srcPortRange, err := networkservice.ParsePortRange(policy.SrcPort)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if srcPortRange != nil {
		rule.Sport = netlink.NewRulePortRange(srcPortRange.Start, srcPortRange.End)
	}
	return rule, nil
}

func ruleAdd(ctx context.Context, handle *netlink.Handle, policy *networkservice.PolicyRoute, tableID int) error {
	rule, err := policyToRule(policy)
	if err != nil {
		return errors.WithStack(err)
	}
	rule.Table = tableID

	now := time.Now()
	if err := handle.RuleAdd(rule); err != nil {
		log.FromContext(ctx).
			WithField("From", policy.From).
			WithField("IPProto", policy.Proto).
			WithField("DstPort", policy.DstPort).
			WithField("SrcPort", policy.SrcPort).
			WithField("Table", tableID).
			WithField("duration", time.Since(now)).
			WithField("netlink", "RuleAdd").Errorf("error %+v", err)
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("From", policy.From).
		WithField("IPProto", policy.Proto).
		WithField("DstPort", policy.DstPort).
		WithField("SrcPort", policy.SrcPort).
		WithField("Table", tableID).
		WithField("duration", time.Since(now)).
		WithField("netlink", "RuleAdd").Debug("completed")
	return nil
}

func defaultRoute() *networkservice.Route {
	return &networkservice.Route{
		Prefix: "0.0.0.0/0",
	}
}

func routeAdd(ctx context.Context, handle *netlink.Handle, l netlink.Link, route *networkservice.Route, tableID int) error {
	if route.GetPrefixIPNet() == nil {
		return errors.New("kernelRoute prefix must not be nil")
	}
	dst := route.GetPrefixIPNet()
	dst.IP = dst.IP.Mask(dst.Mask)
	kernelRoute := &netlink.Route{
		LinkIndex: l.Attrs().Index,
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       dst,
		Table:     tableID,
	}

	gw := route.GetNextHopIP()
	if gw != nil {
		kernelRoute.Gw = gw
		kernelRoute.SetFlag(netlink.FLAG_ONLINK)
	}

	now := time.Now()
	if err := handle.RouteReplace(kernelRoute); err != nil {
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("Dst", kernelRoute.Dst).
			WithField("Gw", kernelRoute.Gw).
			WithField("Scope", kernelRoute.Scope).
			WithField("Flags", kernelRoute.Flags).
			WithField("Table", tableID).
			WithField("duration", time.Since(now)).
			WithField("netlink", "RouteReplace").Errorf("error %+v", err)
		return errors.WithStack(err)
	}
	log.FromContext(ctx).
		WithField("link.Name", l.Attrs().Name).
		WithField("Dst", kernelRoute.Dst).
		WithField("Gw", kernelRoute.Gw).
		WithField("Scope", kernelRoute.Scope).
		WithField("Flags", kernelRoute.Flags).
		WithField("Table", tableID).
		WithField("duration", time.Since(now)).
		WithField("netlink", "RouteReplace").Debug("completed")
	return nil
}

func del(ctx context.Context, conn *networkservice.Connection, tableIDs *Map) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer netlinkHandle.Close()
		ps, ok := tableIDs.LoadAndDelete(conn.GetId())
		if ok {
			for tableID, policy := range ps {
				if err := delRule(ctx, netlinkHandle, policy, tableID); err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}
	return nil
}

func delRule(ctx context.Context, handle *netlink.Handle, policy *networkservice.PolicyRoute, tableID int) error {
	rule, err := policyToRule(policy)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := flushTable(ctx, handle, tableID); err != nil {
		return err
	}

	now := time.Now()
	if err := handle.RuleDel(rule); err != nil {
		log.FromContext(ctx).
			WithField("From", policy.From).
			WithField("IPProto", policy.Proto).
			WithField("DstPort", policy.DstPort).
			WithField("SrcPort", policy.SrcPort).
			WithField("duration", time.Since(now)).
			WithField("netlink", "RuleDel").Errorf("error %+v", err)
		return errors.Wrapf(errors.WithStack(err), "failed to delete rule")
	}
	log.FromContext(ctx).
		WithField("From", policy.From).
		WithField("IPProto", policy.Proto).
		WithField("DstPort", policy.DstPort).
		WithField("SrcPort", policy.SrcPort).
		WithField("duration", time.Since(now)).
		WithField("netlink", "RuleDel").Debug("completed")
	return nil
}

func flushTable(ctx context.Context, handle *netlink.Handle, tableID int) error {
	routes, err := handle.RouteListFiltered(netlink.FAMILY_ALL,
		&netlink.Route{
			Table: tableID,
		},
		netlink.RT_FILTER_TABLE)
	if err != nil {
		return errors.Wrapf(errors.WithStack(err), "failed to list routes")
	}
	for i := 0; i < len(routes); i++ {
		err := handle.RouteDel(&routes[i])
		if err != nil {
			return errors.Wrapf(errors.WithStack(err), "failed to delete route")
		}
	}
	log.FromContext(ctx).
		WithField("tableID", tableID).
		WithField("netlink", "flushTable").Debug("completed")
	return nil
}

func getFreeTableID(ctx context.Context, handle *netlink.Handle) (int, error) {
	routes, err := handle.RouteListFiltered(netlink.FAMILY_ALL,
		&netlink.Route{
			Table: unix.RT_TABLE_UNSPEC,
		},
		netlink.RT_FILTER_TABLE)
	if err != nil {
		return 0, errors.Wrapf(errors.WithStack(err), "getFreeTableID: failed to list routes")
	}

	// tableID = 0 is reserved
	ids := make(map[int]int)
	ids[0] = 0
	for i := 0; i < len(routes); i++ {
		ids[routes[i].Table] = routes[i].Table
	}

	// Find first missing table id
	tableID := len(ids)
	for i := 0; i < len(ids); i++ {
		if _, ok := ids[i]; !ok {
			tableID = i
			break
		}
	}

	log.FromContext(ctx).
		WithField("tableID", tableID).
		WithField("netlink", "getFreeTableID").Debug("completed")

	return tableID, nil
}
