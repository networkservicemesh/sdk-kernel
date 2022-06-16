// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

//go:build perm
// +build perm

package nshandle_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vishvananda/netns"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

const (
	concurrentCount = 50
	equalFormat     = "Not equal: \n" +
		"expected: %v\n" +
		"actual  : %v"
	notEqualFormat = "Should not be: %#v\n"
)

func TestNSHandle_RunInPerm(t *testing.T) {
	goleak.VerifyNone(t)

	current, currErr := nshandle.Current()
	require.NoError(t, currErr)
	defer func() { _ = current.Close() }()

	wg := sync.WaitGroup{}
	wg.Add(concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		go func() {
			defer wg.Done()

			target := newNSHandle(t)
			defer func() { _ = target.Close() }()

			require.False(t, current.Equal(target), notEqualFormat, current, target)

			err := nshandle.RunIn(current, target, func() error {
				handle, err := netns.Get()
				require.NoError(t, err)
				defer func() { _ = handle.Close() }()

				require.True(t, target.Equal(handle), equalFormat, target, handle)

				return nil
			})
			require.NoError(t, err)
		}()
	}

	wg.Wait()
}

func newNSHandle(t *testing.T) netns.NsHandle {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	baseHandle, err := netns.Get()
	require.NoError(t, err)
	defer func() {
		_ = netns.Set(baseHandle)
		_ = baseHandle.Close()
	}()

	newHandle, err := netns.New()
	require.NoError(t, err)

	return newHandle
}
