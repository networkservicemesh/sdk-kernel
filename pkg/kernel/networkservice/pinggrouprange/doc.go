// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package pinggrouprange sets /proc/sys/net/ipv4/ping_group_range variable
//
// We use ping to check the liveliness of the connection. In order not to use the root privileges, we need to set the
// ping_group_range value. It allows to create the SOCK_DGRAM socket type (instead of SOCK_RAW) for ping and thus use it
// in non-privileged mode.
// See:
// https://github.com/go-ping/ping
// https://www.kernel.org/doc/Documentation/networking/ip-sysctl.txt
package pinggrouprange
