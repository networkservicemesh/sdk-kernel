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

func TestInjectLinkInNamespace(t *testing.T) {
  g := NewWithT(t)

  // set up the basic resources for the test
  currns, _ := netns.Get()
  testns, _ := netns.New()
  _ = netns.Set(currns)
  vethErr := netlink.LinkAdd(kernelutils.NewVETH(testVethSrcName, testVethDstName))
  g.Expect(vethErr).To(BeNil())

  // clean up after ourselves
  defer func(rootns netns.NsHandle, testns netns.NsHandle, veth *netlink.Veth) {
    _ = netns.Set(rootns)
    testns.Close()
    netlink.LinkDel(veth)
  }(currns, testns, kernelutils.NewVETH(testVethSrcName, testVethDstName))

  // move one end of the veth into testns and run assertions
  injectError := kernelutils.InjectLinkInNamespace(testns, testVethDstName)

  _ = netns.Set(testns)
  nsLinks, _ := netlink.LinkList()
  _, linkErr := netlink.LinkByName(testVethDstName)

  // assert no error on link injection
  g.Expect(injectError).To(BeNil())
  // assert only loopback and the interface we inject exist in the namespace
  g.Expect(len(nsLinks)).To(Equal(2))
  g.Expect(linkErr).To(BeNil())

	logrus.Printf("Done")
}
