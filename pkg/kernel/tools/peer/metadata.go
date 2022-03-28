// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// Package peer allows storing peer netlink.Link in per Connection.Id metadata
package peer

import (
	"context"

	"github.com/vishvananda/netlink"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// Store sets the netlink.Link stored in per Connection.Id metadata.
func Store(ctx context.Context, isClient bool, link netlink.Link) {
	metadata.Map(ctx, isClient).Store(key{}, link)
}

// Delete deletes the netlink.Link stored in per Connection.Id metadata
func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// Load returns the netlink.Link stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context, isClient bool) (value netlink.Link, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(netlink.Link)
	return value, ok
}

// LoadOrStore returns the existing netlink.Link stored in per Connection.Id metadata if present.
// Otherwise, it stores and returns the given nterface_types.InterfaceIndex.
// The loaded result is true if the value was loaded, false if stored.
func LoadOrStore(ctx context.Context, isClient bool, link netlink.Link) (value netlink.Link, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{}, link)
	if !ok {
		return
	}
	value, ok = rawValue.(netlink.Link)
	return value, ok
}

// LoadAndDelete deletes the netlink.Link stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func LoadAndDelete(ctx context.Context, isClient bool) (value netlink.Link, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(netlink.Link)
	return value, ok
}
