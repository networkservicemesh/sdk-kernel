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

package sdk_kernel_tests

import (
	"testing"

  . "github.com/onsi/gomega"

  "github.com/networkservicemesh/sdk-kernel/pkg/networkservice/kernelutils"

  "github.com/sirupsen/logrus"
  "github.com/vishvananda/netlink"
  "github.com/vishvananda/netns"
)

const testVethSrcName = "sdk-kernel-src"
const testVethDstName = "sdk-kernel-dst"
var testVeth = netlink.Veth {
	LinkAttrs: netlink.LinkAttrs {
		Name: testVethSrcName,
	},
	PeerName: testVethDstName,
}

func TestInjectLinkInNamespace(t *testing.T) {
  g := NewWithT(t)

  // set up the basic resources for the test
  currns, _ := netns.Get()
  testns, _ := netns.New()
  _ = netns.Set(currns)
  vethErr := netlink.LinkAdd(&testVeth)
  g.Expect(vethErr).To(BeNil(), "Error creating test veth pair: %v", vethErr)

  // clean up after ourselves
  defer func(rootns netns.NsHandle, testns netns.NsHandle, veth *netlink.Veth) {
    _ = netns.Set(rootns)
    testns.Close()
    netlink.LinkDel(veth)
  }(currns, testns, &testVeth)

  // move one end of the veth into testns and run assertions
  injectError := kernelutils.InjectLinkInNamespace(testns, testVeth.PeerName)

  _ = netns.Set(testns)
  nsLinks, _ := netlink.LinkList()
  _, linkErr := netlink.LinkByName(testVeth.PeerName)

  // assert no error on link injection
  g.Expect(injectError).To(BeNil(), "Error injecting test interface in namespace: %v", injectError)
  // assert only loopback and the interface we inject exist in the namespace
  g.Expect(len(nsLinks)).To(Equal(2))
  g.Expect(linkErr).To(BeNil(), "Error getting locating test interface in namespace: %v", linkErr)

	logrus.Printf("Done")
}

func TestIdempotentInjectLinkInNamespace(t *testing.T) {
  g := NewWithT(t)

  // set up the basic resources for the test
  currns, _ := netns.Get()
  testns, _ := netns.New()
  _ = netns.Set(currns)
  vethErr := netlink.LinkAdd(&testVeth)
  g.Expect(vethErr).To(BeNil(), "Error creating test veth pair: %v", vethErr)

  // clean up after ourselves
  defer func(rootns netns.NsHandle, testns netns.NsHandle, veth *netlink.Veth) {
    _ = netns.Set(rootns)
    testns.Close()
    netlink.LinkDel(veth)
  }(currns, testns, &testVeth)

  // move one end of the veth into testns and run assertions
  injectError := kernelutils.InjectLinkInNamespace(testns, testVeth.PeerName)
	// assert no error on initial link injection
  g.Expect(injectError).To(BeNil(), "Error injecting test interface in namespace: %v", injectError)

  // attempt to re-inject the link in the target namespace
	injectError = kernelutils.InjectLinkInNamespace(testns, testVeth.PeerName)
  _ = netns.Set(testns)
  nsLinks, _ := netlink.LinkList()
  _, linkErr := netlink.LinkByName(testVeth.PeerName)

  // assert no error on link injection
  g.Expect(injectError).To(BeNil(), "Error injecting test interface in namespace: %v", injectError)
  // assert only loopback and the interface we inject exist in the namespace
  g.Expect(len(nsLinks)).To(Equal(2))
  g.Expect(linkErr).To(BeNil(), "Error getting locating test interface in namespace: %v", linkErr)

	logrus.Printf("Done")
}
