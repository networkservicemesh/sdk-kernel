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

// Package nsswitch contains tool to switch between net namespaces
package nsswitch

import (
	"runtime"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
)

// NSSwitch is a tool to switch between net namespaces
type NSSwitch struct {
	// NetNSHandle is a base net namespace handle
	NetNSHandle netns.NsHandle
}

// NewNSSwitch returns a new NSSwitch
func NewNSSwitch() (s *NSSwitch, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s = &NSSwitch{}
	if s.NetNSHandle, err = netns.Get(); err != nil {
		return nil, err
	}

	return s, nil
}

// NewNSSwitchAndHandle returns a new NSSwitch and a net NS handle from the given URL
func NewNSSwitchAndHandle(netNSURL string) (nsSwitch *NSSwitch, netNSHandle netns.NsHandle, err error) {
	nsSwitch, err = NewNSSwitch()
	if err != nil {
		return nil, -1, errors.Wrap(err, "failed to init net NS switch")
	}

	netNSHandle, err = netns.GetFromPath(netNSURL)
	if err != nil {
		_ = nsSwitch.Close()
		return nil, -1, errors.Wrapf(err, "failed to obtain network NS handle")
	}

	return nsSwitch, netNSHandle, nil
}

// RunIn runs runner in the given net NS
func (s *NSSwitch) RunIn(netNSHandle netns.NsHandle, runner func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	switch currNetNSHandle, err := netns.Get(); {
	case err != nil:
		return err
	case !currNetNSHandle.Equal(s.NetNSHandle):
		return errors.Errorf("current net NS is not the switch net NS: %v != %v", netNSHandle, s.NetNSHandle)
	}

	if !netNSHandle.Equal(s.NetNSHandle) {
		if err := netns.Set(netNSHandle); err != nil {
			return errors.Wrapf(err, "failed to switch to the runner net NS: %v", netNSHandle)
		}
		defer func() {
			if err := netns.Set(s.NetNSHandle); err != nil {
				panic(errors.Wrapf(err, "failed to switch back to the switch net NS: %v", s.NetNSHandle).Error())
			}
		}()
	}

	return runner()
}

// Close closes the handle opened by NSSwitch
func (s *NSSwitch) Close() error {
	if err := s.NetNSHandle.Close(); err != nil {
		return err
	}
	s.NetNSHandle = -1

	return nil
}
