// Copyright (c) 2026 Nordix Foundation.
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

//go:build perm && linux

package kernel

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	testIfName    = "nsmtest0"
	testIterCount = 25
)

// countSocketFDs returns the number of file descriptors of the current process
// that point at a socket (the `socket:[inode]` links under /proc/self/fd).
// A netlink handle holds one socket per netlink family.
func countSocketFDs(t *testing.T) int {
	entries, err := os.ReadDir("/proc/self/fd")
	require.NoError(t, err)

	var count int
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			// fd was closed while we were walking the directory
			continue
		}
		if strings.HasPrefix(target, "socket:[") {
			count++
		}
	}
	return count
}

// newNetNSWithLink creates a fresh net NS holding a single dummy interface,
// emulating the net NS FindHostDevice searches in.
func newNetNSWithLink(t *testing.T) netns.NsHandle {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	baseHandle, err := netns.Get()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, netns.Set(baseHandle))
		_ = baseHandle.Close()
	}()

	targetHandle, err := netns.New()
	require.NoError(t, err)

	handle, err := netlink.NewHandleAt(targetHandle)
	require.NoError(t, err)
	defer handle.Close()

	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: testIfName}}
	require.NoError(t, handle.LinkAdd(dummy))

	return targetHandle
}

// TestFindHostDeviceFDLeakPerm - FindHostDevice() must not leak the netlink
// handle it creates while searching for the interface by name. The inject chain
// element calls it on every Request and Close, so a leaked handle per call
// accumulates netlink sockets, and each of them keeps a reference to the net NS
// it was opened in.
func TestFindHostDeviceFDLeakPerm(t *testing.T) {
	targetNetNS := newNetNSWithLink(t)
	defer func() { _ = targetNetNS.Close() }()

	// Warm up, so that lazily initialized state is not counted as a leak.
	l, err := FindHostDevice("", testIfName, targetNetNS)
	require.NoError(t, err)
	require.Equal(t, testIfName, l.GetName())

	fdsBefore := countSocketFDs(t)

	for i := 0; i < testIterCount; i++ {
		l, err = FindHostDevice("", testIfName, targetNetNS)
		require.NoError(t, err)
		require.Equal(t, testIfName, l.GetName())
	}

	fdsAfter := countSocketFDs(t)
	require.Equal(t, fdsBefore, fdsAfter,
		"FindHostDevice() leaked %d socket fd(s) over %d calls", fdsAfter-fdsBefore, testIterCount)
}
