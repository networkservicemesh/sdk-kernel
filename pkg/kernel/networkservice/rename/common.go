// Copyright (c) 2021 Nordix Foundation.
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

package rename

import (
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
)

func renameLink(oldName, newName string) error {
	link, err := netlink.LinkByName(oldName)
	if err != nil {
		return errors.Wrapf(err, "failed to get the net interface: %v", oldName)
	}

	if err = netlink.LinkSetName(link, newName); err != nil {
		return errors.Wrapf(err, "failed to rename net interface: %v -> %v", oldName, newName)
	}

	return nil
}
