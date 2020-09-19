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

import "context"

const (
	configKey key = "vfconfig.VFConfig"
)

type key string

// VFConfig is a config for VF
type VFConfig struct {
	// PFInterfaceName is a parent PF net interface name
	PFInterfaceName string
	// VFInterfaceName is a VF net interface name
	VFInterfaceName string
	// VFNum is a VF num for the parent PF
	VFNum int
}

// WithConfig returns new context with VFConfig
func WithConfig(parent context.Context, config *VFConfig) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, configKey, config)
}

// Config returns VFConfig from context
func Config(ctx context.Context) *VFConfig {
	if rv, ok := ctx.Value(configKey).(*VFConfig); ok {
		return rv
	}
	return nil
}
