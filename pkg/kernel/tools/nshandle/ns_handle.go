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

// Package nshandle provides utils for creating net NS handles
package nshandle

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/vishvananda/netns"
)

// FromURL creates net NS handle by file://path URL
func FromURL(urlString string) (handle netns.NsHandle, err error) {
	var netNSURL *url.URL
	netNSURL, err = url.Parse(urlString)
	if err != nil || netNSURL.Scheme != "file" {
		return -1, errors.Wrapf(err, "invalid url: %v", urlString)
	}

	handle, err = netns.GetFromPath(netNSURL.Path)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to obtain network NS handle")
	}

	return handle, nil
}
