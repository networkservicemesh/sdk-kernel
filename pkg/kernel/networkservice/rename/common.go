// Copyright (C) 2021, Nordix Foundation
//
// All rights reserved. This program and the accompanying materials
// are made available under the terms of the Apache License, Version 2.0
// which accompanies this distribution, and is available at
// http://www.apache.org/licenses/LICENSE-2.0

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
