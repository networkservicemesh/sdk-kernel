// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

package routes

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		netlinkHandle, err := link.GetNetlinkHandle(mechanism.GetNetNSURL())
		if err != nil {
			return errors.WithStack(err)
		}
		defer netlinkHandle.Close()

		ifName := mechanism.GetInterfaceName()

		l, err := netlinkHandle.LinkByName(ifName)
		if err != nil {
			return errors.WithStack(err)
		}

		if err = netlinkHandle.LinkSetUp(l); err != nil {
			return errors.WithStack(err)
		}

		var linkRoutes []*networkservice.Route
		var routes []*networkservice.Route
		if isClient {
			linkRoutes = conn.GetContext().GetIpContext().GetSrcIPRoutes()
			routes = conn.GetContext().GetIpContext().GetDstRoutesWithExplicitNextHop()
		} else {
			linkRoutes = conn.GetContext().GetIpContext().GetDstIPRoutes()
			routes = conn.GetContext().GetIpContext().GetSrcRoutesWithExplicitNextHop()
		}
		for _, route := range linkRoutes {
			if err := routeAdd(ctx, netlinkHandle, l, netlink.SCOPE_LINK, route); err != nil {
				return err
			}
		}
		for _, route := range routes {
			if err := routeAdd(ctx, netlinkHandle, l, netlink.SCOPE_UNIVERSE, route); err != nil {
				return err
			}
		}
	}
	return nil
}

func routeAdd(ctx context.Context, handle *netlink.Handle, l netlink.Link, scope netlink.Scope, route *networkservice.Route) error {
	if route.GetPrefixIPNet() == nil {
		return errors.New("kernelRoute prefix must not be nil")
	}
	dst := route.GetPrefixIPNet()
	dst.IP = dst.IP.Mask(dst.Mask)
	kernelRoute := &netlink.Route{
		LinkIndex: l.Attrs().Index,
		Scope:     scope,
		Dst:       dst,
	}
	gw := route.GetNextHopIP()
	if gw != nil {
		kernelRoute.Gw = gw
		if scope != netlink.SCOPE_LINK {
			kernelRoute.SetFlag(netlink.FLAG_ONLINK)
		}
	}

	now := time.Now()
	if err := handle.RouteReplace(kernelRoute); err != nil {
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("Dst", kernelRoute.Dst).
			WithField("Gw", kernelRoute.Gw).
			WithField("Scope", kernelRoute.Scope).
			WithField("Flags", kernelRoute.Flags).
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
		WithField("duration", time.Since(now)).
		WithField("netlink", "RouteReplace").Debug("completed")
	return nil
}
