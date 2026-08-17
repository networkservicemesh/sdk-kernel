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

package iprule

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/edwarnicke/genericsync"
	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
)

const (
	testIfName    = "nsmtest0"
	testConnID    = "conn-1"
	testIterCount = 25
)

// countNetNSFDs returns the number of file descriptors of the current process
// that point at a network namespace (the `net:[inode]` links under /proc/self/fd).
func countNetNSFDs(t *testing.T) int {
	entries, err := os.ReadDir("/proc/self/fd")
	require.NoError(t, err)

	var count int
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if err != nil {
			// fd was closed while we were walking the directory
			continue
		}
		if strings.HasPrefix(target, "net:[") {
			count++
		}
	}
	return count
}

// newTargetNetNS creates a fresh net NS, emulating the net NS of a client pod.
func newTargetNetNS(t *testing.T) netns.NsHandle {
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

	return targetHandle
}

// newTestConn returns a connection with a kernel mechanism pointing at the given
// net NS and a single policy route, and creates the interface the policy applies to.
//
// The net NS is addressed as file:///proc/${pid}/fd/${fd}, which is the form the
// NSM forwarder receives from the NSC.
func newTestConn(t *testing.T, targetNetNS netns.NsHandle) *networkservice.Connection {
	handle, err := netlink.NewHandleAt(targetNetNS)
	require.NoError(t, err)
	defer handle.Close()

	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: testIfName}}
	require.NoError(t, handle.LinkAdd(dummy))
	require.NoError(t, handle.LinkSetUp(dummy))

	return &networkservice.Connection{
		Id: testConnID,
		Mechanism: &networkservice.Mechanism{
			Cls:  cls.LOCAL,
			Type: kernelmech.MECHANISM,
			Parameters: map[string]string{
				kernelmech.NetNSURL:         fmt.Sprintf("file:///proc/%d/fd/%d", os.Getpid(), int(targetNetNS)),
				kernelmech.InterfaceNameKey: testIfName,
			},
		},
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				Policies: []*networkservice.PolicyRoute{
					{
						From:    "10.0.0.1/32",
						DstPort: "8080",
					},
				},
			},
		},
	}
}

// TestIPRuleFDLeakOnRefreshPerm - create() must not leak a net NS fd per Request.
// create() is called on every Request, including every refresh of an established
// connection, so a single leaked fd per call exhausts the forwarder's fd limit.
func TestIPRuleFDLeakOnRefreshPerm(t *testing.T) {
	targetNetNS := newTargetNetNS(t)
	defer func() { _ = targetNetNS.Close() }()

	conn := newTestConn(t, targetNetNS)
	tableIDs := new(genericsync.Map[string, policies])
	nsRTableNextIDToConnID := new(genericsync.Map[netnsRTableNextID, string])

	// First Request - installs the policy and any long lived state.
	require.NoError(t, create(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))

	fdsBefore := countNetNSFDs(t)

	// Refreshes - nothing to add or remove, but the net NS is opened every time.
	for i := 0; i < testIterCount; i++ {
		require.NoError(t, create(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
	}

	fdsAfter := countNetNSFDs(t)
	require.Equal(t, fdsBefore, fdsAfter,
		"create() leaked %d net NS fd(s) over %d refreshes", fdsAfter-fdsBefore, testIterCount)
}

// TestIPRuleFDLeakOnClosePerm - del() must not leak a net NS fd per Close.
func TestIPRuleFDLeakOnClosePerm(t *testing.T) {
	targetNetNS := newTargetNetNS(t)
	defer func() { _ = targetNetNS.Close() }()

	conn := newTestConn(t, targetNetNS)
	tableIDs := new(genericsync.Map[string, policies])
	nsRTableNextIDToConnID := new(genericsync.Map[netnsRTableNextID, string])

	// Warm up one full cycle.
	require.NoError(t, create(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
	require.NoError(t, del(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))

	fdsBefore := countNetNSFDs(t)

	for i := 0; i < testIterCount; i++ {
		require.NoError(t, create(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
		require.NoError(t, del(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
	}

	fdsAfter := countNetNSFDs(t)
	require.Equal(t, fdsBefore, fdsAfter,
		"create()/del() leaked %d net NS fd(s) over %d cycles", fdsAfter-fdsBefore, testIterCount)
}

// TestIPRuleFDLeakOnRecoverPerm - recoverTableIDs()/deleteRemainders() must not
// leak a net NS fd. This path is taken on the first Request after a forwarder
// restart, and on every Request for a connection whose table IDs are not cached.
func TestIPRuleFDLeakOnRecoverPerm(t *testing.T) {
	targetNetNS := newTargetNetNS(t)
	defer func() { _ = targetNetNS.Close() }()

	conn := newTestConn(t, targetNetNS)
	tableIDs := new(genericsync.Map[string, policies])
	nsRTableNextIDToConnID := new(genericsync.Map[netnsRTableNextID, string])

	require.NoError(t, create(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
	// Drop the cache, so the next Request has to recover the table IDs.
	tableIDs.Delete(testConnID)
	require.NoError(t, recoverTableIDs(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))

	fdsBefore := countNetNSFDs(t)

	for i := 0; i < testIterCount; i++ {
		tableIDs.Delete(testConnID)
		require.NoError(t, recoverTableIDs(context.Background(), conn, tableIDs, nsRTableNextIDToConnID))
	}

	fdsAfter := countNetNSFDs(t)
	require.Equal(t, fdsBefore, fdsAfter,
		"recoverTableIDs() leaked %d net NS fd(s) over %d calls", fdsAfter-fdsBefore, testIterCount)
}
