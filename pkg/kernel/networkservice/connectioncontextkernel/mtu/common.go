// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package mtu

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"

	link "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

func setMTU(ctx context.Context, conn *networkservice.Connection) error {
	if mechanism := kernel.ToMechanism(conn.GetMechanism()); mechanism != nil && mechanism.GetVLAN() == 0 {
		// Note: These are switched from normal because if we are the client, we need to assign the IP
		// in the Endpoints NetNS for the Dst.  If we are the *server* we need to assign the IP for the
		// clients NetNS (ie the source).
		mtu := conn.GetContext().GetMTU()
		if mtu == 0 {
			return nil
		}

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

		now := time.Now()
		if err := netlinkHandle.LinkSetMTU(l, int(mtu)); err != nil {
			return errors.Wrapf(err, "error attempting to set MTU on link %q to value %q", l.Attrs().Name, mtu)
		}
		log.FromContext(ctx).
			WithField("link.Name", l.Attrs().Name).
			WithField("MTU", mtu).
			WithField("duration", time.Since(now)).
			WithField("netlink", "LinkSetMTU").Debug("completed")
	}
	return nil
}
