// Copyright 2019 VMware, Inc.
// Copyright 2020 SUSE LLC.
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

package kernelutils

import (
  "github.com/sirupsen/logrus"
  "github.com/vishvananda/netlink"
  "github.com/vishvananda/netns"
)

func InjectLinkInNamespace(targetNs netns.NsHandle, ifaceName string) error {
  hostNs, nsErr := netns.Get()
  defer hostNs.Close()
  if nsErr != nil {
    logrus.Errorf("sdk-kernel: failed to get host namespace")
    return nsErr
  }

  /* Get a link object for the interface */
  ifaceLink, linkErr := netlink.LinkByName(ifaceName)
  if linkErr != nil {
    // link was not found in host namespace, see if we can find it in target namespace
    nsErr = netns.Set(targetNs)
    defer netns.Set(hostNs)
    if nsErr != nil {
		  logrus.Errorf("sdk-kernel: failed switching to desired namespace: %v", nsErr)
		  return nsErr
	  }
    _, linkErr := netlink.LinkByName(ifaceName)
    if linkErr != nil {
      // we didn't find the link in the target namespace either, return an error
      return linkErr
    }
    nsErr = netns.Set(hostNs)
    if nsErr != nil {
		  logrus.Errorf("sdk-kernel: failed switching to host namespace: %v", nsErr)
		  return nsErr
	  }
  } else {
    /* Inject the interface into the desired namespace */
    linkErr = netlink.LinkSetNsFd(ifaceLink, int(targetNs))
    if linkErr != nil {
      logrus.Errorf("sdk-kernel: failed to inject %q in namespace - %v", ifaceName, linkErr)
      return linkErr
    }
  }
  return nil
}
