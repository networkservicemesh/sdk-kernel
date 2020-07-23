// Copyright (c) 2020 Intel Corporation. All Rights Reserved.
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

// Package kernel contains Link representation of network interface
package kernel

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// State defines admin state of the network interface
type State uint

const (
	// DOWN is link admin state down
	DOWN State = iota
	// UP is link admin state down
	UP
)

// Link represents network interface and specifies operations
// that can be performed on that interface.
type Link interface {
	AddAddress(ip string) error
	DeleteAddress(ip string) error
	MoveToNetns(target netns.NsHandle) error
	SetAdminState(state State) error
	SetName(name string) error
}

// link provides Link interface implementation
type link struct {
	link  netlink.Link
	netns netns.NsHandle
}

func (l *link) MoveToNetns(target netns.NsHandle) error {
	// don't do anything if already there
	if l.netns.Equal(target) {
		return nil
	}

	// set link down
	err := l.SetAdminState(DOWN)
	if err != nil {
		return errors.Errorf("failed to move link %s to netns: %q", l.link, err)
	}

	// set netns
	err = netlink.LinkSetNsFd(l.link, int(target))
	if err != nil {
		return errors.Errorf("failed to move link %s to netns: %q", l.link, err)
	}

	l.netns = target

	return nil
}

func (l *link) AddAddress(ip string) error {
	// parse IP address
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return errors.Errorf("failed to parse IP address %q: %s", ip, err)
	}

	// check if address is already assigned
	current, err := netlink.AddrList(l.link, netlink.FAMILY_ALL)
	if err != nil {
		return errors.Errorf("failed to get current IP address list %q: %s", ip, err)
	}

	for _, existing := range current {
		if addr.Equal(existing) {
			// nothing to do
			return nil
		}
	}

	// add address
	err = netlink.AddrAdd(l.link, addr)
	if err != nil {
		return errors.Errorf("failed to add IP address %q: %s", ip, err)
	}

	return nil
}

func (l *link) DeleteAddress(ip string) error {
	// parse IP address
	addr, err := netlink.ParseAddr(ip)
	if err != nil {
		return errors.Errorf("failed to parse IP address %q: %s", ip, err)
	}

	// delete address
	err = netlink.AddrDel(l.link, addr)
	if err != nil {
		return errors.Errorf("failed to delete IP address %q: %s", ip, err)
	}

	return nil
}

func (l *link) SetAdminState(state State) error {
	switch state {
	case DOWN:
		err := netlink.LinkSetDown(l.link)
		if err != nil {
			return errors.Errorf("failed to set %s down: %s", l.link, err)
		}
	case UP:
		err := netlink.LinkSetUp(l.link)
		if err != nil {
			return errors.Errorf("failed to bring %s up: %s", l.link, err)
		}
	}

	return nil
}

func (l *link) SetName(name string) error {
	if l.link.Attrs().Name != name {
		err := netlink.LinkSetName(l.link, name)
		if err != nil {
			return errors.Errorf("failed to set interface name to %s: %s", name, err)
		}
	}

	return nil
}

// FindHostDevice returns a new instance of link representing host device, based on the PCI
// address and/or target interface name.
func FindHostDevice(pciAddress, name string, namespaces ...netns.NsHandle) (Link, error) {
	// TODO: add support for shared l interfaces (like Mellanox NICs)

	attempts := []func(netns.NsHandle, string, string) (netlink.Link, error){
		searchByPCIAddress,
		searchByName,
	}

	// search for link with a matching name or PCI address in the provided namespaces
	for _, ns := range namespaces {
		for _, search := range attempts {
			found, err := search(ns, name, pciAddress)
			if err != nil {
				continue
			}

			if found != nil {
				return &link{
					link:  found,
					netns: ns,
				}, nil
			}
		}
	}

	return nil, errors.Errorf("failed to obtain netlink link matching criteria: name=%s or pciAddress=%s", name, pciAddress)
}

func searchByPCIAddress(ns netns.NsHandle, name, pciAddress string) (netlink.Link, error) {
	// execute in context of the pod's namespace
	err := netns.Set(ns)
	if err != nil {
		return nil, errors.Errorf("failed to enter namespace: %s", err)
	}

	netDir := filepath.Join("/sys/bus/pci/devices", pciAddress, "net")
	if _, err = os.Lstat(netDir); err != nil {
		return nil, errors.Errorf("no net directory under pci device %s: %q", pciAddress, err)
	}

	fInfos, err := ioutil.ReadDir(netDir)
	if err != nil {
		return nil, errors.Errorf("failed to read net directory %s: %q", netDir, err)
	}

	names := make([]string, 0)
	for _, f := range fInfos {
		names = append(names, f.Name())
	}

	if len(names) == 0 {
		return nil, errors.Errorf("no links with PCI address %s found", pciAddress)
	}

	link, err := netlink.LinkByName(names[0])
	if err != nil {
		return nil, errors.Errorf("error getting host device with PCI address %s", pciAddress)
	}

	return link, nil
}

func searchByName(ns netns.NsHandle, name, pciAddress string) (netlink.Link, error) {
	// execute in context of the pod's namespace
	err := netns.Set(ns)
	if err != nil {
		return nil, errors.Errorf("failed to switch to namespace: %s", err)
	}

	// get link
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, errors.Errorf("failed to get link with name %s", name)
	}

	return link, nil
}
