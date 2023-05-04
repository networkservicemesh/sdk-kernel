// Copyright (c) 2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Nordix Foundation.
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

package iprule

import (
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

//go:generate go-syncmap -output table_map.gen.go -type Map<string,policies>
//go:generate go-syncmap -output netns_rtable_conn_map.gen.go -type NetnsRTableNextIDToConnMap<NetnsRTableNextID,string>

type policies map[int]*networkservice.PolicyRoute

// Map - sync.Map with key == string (connID) and value == policies
type Map sync.Map

// NetnsRTableNextIDToConnMap - sync.Map with key NetnsRTableNextID value of string (connID)
type NetnsRTableNextIDToConnMap sync.Map
