// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package utils

import (
	"runtime"

	"github.com/vishvananda/netns"
)

// NSSwitch provides methods to switch between net namespaces
type NSSwitch struct {
	// NetNSHandle is a base net namespace handle
	NetNSHandle  netns.NsHandle
	netNSHandles []netns.NsHandle
}

// NewNSSwitch returns a new NSSwitch
func NewNSSwitch() (s *NSSwitch, err error) {
	s = &NSSwitch{}

	s.Lock()
	defer s.Unlock()

	if s.NetNSHandle, err = netns.Get(); err != nil {
		return nil, err
	}

	return s, nil
}

// SwitchByNetNSHandle switches net namespace by handle
func (s *NSSwitch) SwitchByNetNSHandle(netNSHandle netns.NsHandle) error {
	currNetNSHandle, err := netns.Get()
	if err != nil {
		return err
	}

	if currNetNSHandle.Equal(netNSHandle) {
		return nil
	}
	return netns.Set(netNSHandle)
}

// SwitchByNetNSInode switches net namespace by inode
func (s *NSSwitch) SwitchByNetNSInode(netNSInode string) error {
	netNSHandle, err := GetNSHandleFromInode(netNSInode)
	if err != nil {
		return err
	}
	s.netNSHandles = append(s.netNSHandles, netNSHandle)

	return s.SwitchByNetNSHandle(netNSHandle)
}

// Lock locks OS thread
func (s *NSSwitch) Lock() {
	runtime.LockOSThread()
}

// Unlock unlocks OS thread
func (s *NSSwitch) Unlock() {
	runtime.UnlockOSThread()
}

// Close closes all handles opened by NSSwitch
func (s *NSSwitch) Close() error {
	for _, netNSHandle := range s.netNSHandles {
		_ = netNSHandle.Close()
	}
	return s.NetNSHandle.Close()
}
