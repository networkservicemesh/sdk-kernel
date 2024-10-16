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

package pinggrouprange

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

// See https://github.com/go-ping/ping#linux
const (
	pingGroupRangeFilename = "/proc/sys/net/ipv4/ping_group_range"
	groupRange             = "0 2147483647"
)

func applyPingGroupRange(ctx context.Context, mech *kernel.Mechanism) error {
	forwarderNetNS, err := nshandle.Current()
	if err != nil {
		return err
	}
	defer func() { _ = forwarderNetNS.Close() }()

	var targetNetNS netns.NsHandle
	targetNetNS, err = nshandle.FromURL(mech.GetNetNSURL())
	if err != nil {
		return err
	}
	defer func() { _ = targetNetNS.Close() }()

	if err = nshandle.RunIn(forwarderNetNS, targetNetNS, func() error {
		return os.WriteFile(pingGroupRangeFilename, []byte(groupRange), 0o600)
	}); err != nil {
		return errors.Wrapf(err, "failed to set %s = %s", pingGroupRangeFilename, groupRange)
	}
	log.FromContext(ctx).Debugf("%s was set to %s", pingGroupRangeFilename, groupRange)
	return nil
}
