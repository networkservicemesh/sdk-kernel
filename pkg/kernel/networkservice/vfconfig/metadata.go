// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

// Package vfconfig provides VF config
package vfconfig

import (
	"context"

	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// VFConfig is a config for VF
type VFConfig struct {
	// PFInterfaceName is a parent PF net interface name
	PFInterfaceName string
	// VFInterfaceName is a VF net interface name
	VFInterfaceName string
	// VFPCIAddress is a VF pci address
	VFPCIAddress string
	// VFNum is a VF num for the parent PF
	VFNum int
	// ContNetNS is a container netns id on which VF is attached
	ContNetNS netns.NsHandle
}

// Store sets the VFConfig stored in per Connection.Id metadata.
func Store(ctx context.Context, isClient bool, config *VFConfig) {
	metadata.Map(ctx, isClient).Store(key{}, config)
}

// Delete deletes the VFConfig stored in per Connection.Id metadata
func Delete(ctx context.Context, isClient bool) {
	metadata.Map(ctx, isClient).Delete(key{})
}

// Load returns the VFConfig stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context, isClient bool) (config *VFConfig, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(key{})
	if !ok {
		return
	}
	config, ok = rawValue.(*VFConfig)
	return config, ok
}

// LoadOrStore returns the existing VFConfig stored in per Connection.Id metadata if present.
// Otherwise, it stores and returns the given VFConfig.
// The loaded result is true if the value was loaded, false if stored.
func LoadOrStore(ctx context.Context, isClient bool, config *VFConfig) (value *VFConfig, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadOrStore(key{}, config)
	if !ok {
		return config, ok
	}
	value, ok = rawValue.(*VFConfig)
	return value, ok
}

// LoadAndDelete deletes the VFConfig stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func LoadAndDelete(ctx context.Context, isClient bool) (config *VFConfig, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).LoadAndDelete(key{})
	if !ok {
		return
	}
	config, ok = rawValue.(*VFConfig)
	return config, ok
}
