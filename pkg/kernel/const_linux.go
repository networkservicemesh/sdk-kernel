// Copyright (c) 2022 Cisco and/or its affiliates.
//
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

//go:build linux
// +build linux

package kernel

import (
	"github.com/vishvananda/netlink"
)

const (
	// FamilyAll is netlink.FAMILY_ALL
	FamilyAll = netlink.FAMILY_ALL
	// NudReachable is netlink.NUD_REACHABLE
	NudReachable = netlink.NUD_REACHABLE
)
