package netnsconnectioncontext

import (
	"context"
	"net"
	"os"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel"
	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

type netnsConnectionContextServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &netnsConnectionContextServer{}
}

func (n *netnsConnectionContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	if err := addIPs(ctx, request.GetConnection(), metadata.IsClient(n)); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (n *netnsConnectionContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func addIPs(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		nsFilename := mechanism.GetNetNSURL()
		hostIfName := mechanism.GetInterfaceName(conn)

		logger := log.FromContext(ctx).WithField("netnsconnectioncontext", "AddIPs").
			WithField("HostIfName", hostIfName).
			WithField("HostNamespace", nsFilename).
			WithField("isClient", isClient)
		logger.Debug("request")

		if nsFilename == "" {
			return nil
		}

		var clientNetNS netns.NsHandle
		clientNetNS, err := nshandle.FromURL(nsFilename)
		if err != nil {
			return errors.Wrapf(err, "handle can not get for client namespace %s", nsFilename)
		}
		defer func() { _ = clientNetNS.Close() }()

		var currNetNS netns.NsHandle
		currNetNS, err = nshandle.Current()
		if err != nil {
			return errors.Wrap(err, "handle can not get for current namespace")
		}
		defer func() { _ = currNetNS.Close() }()

		ipContext := conn.GetContext().GetIpContext()

		ipNet := ipContext.GetDstIpAddrs()
		routes := ipContext.GetDstRoutes()
		if isClient {
			ipNet = ipContext.GetSrcIpAddrs()
			routes = ipContext.GetSrcRoutes()
		}

		logger.Debugf("Is to set IPs: %v and routes: %+v", ipNet, routes)
		return setIPandRoutes(hostIfName, routes, ipNet, currNetNS, clientNetNS)

	}
	return nil
}

func setIPandRoutes(ifName string, routes []*networkservice.Route, ipNet []string, currNetNS, toNetNS netns.NsHandle) error {
	return nshandle.RunIn(currNetNS, toNetNS, func() error {
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			return nil
			//return errors.Wrapf(err, "no link created with name %s", ifName)
		}

		for _, ipN := range ipNet {
			ipAddr, err := netlink.ParseAddr(ipN)
			if err != nil {
				return errors.Wrapf(err, "invalid IP address: %v", ipNet)
			}

			ipAddrs, err := netlink.AddrList(link, kernel.FamilyAll)
			if err != nil {
				return errors.Wrapf(err, "failed to get the net interface IP addresses: %v", link.Attrs().Name)
			}

			for i := range ipAddrs {
				if ipAddr.Equal(ipAddrs[i]) {
					return nil
				}
			}

			if err := netlink.AddrAdd(link, ipAddr); err != nil {
				return errors.Wrapf(err, "failed to add IP address to the net interface: %v %v", link.Attrs().Name, ipAddr)
			}
		}

		for _, route := range routes {
			_, routeNet, err := net.ParseCIDR(route.GetPrefix())
			if err != nil {
				return errors.Wrapf(err, "invalid route CIDR: %v", route.GetPrefix())
			}
			kernelRoute := &netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst: &net.IPNet{
					IP:   routeNet.IP,
					Mask: routeNet.Mask,
				},
			}
			if route.GetNextHopIP() != nil {
				kernelRoute.Gw = route.GetNextHopIP()
			}
			if err = netlink.RouteAdd(kernelRoute); err != nil && !os.IsExist(err) {
				return errors.Wrapf(err, "failed to add route: %v", route.GetPrefix())
			}
		}
		return nil
	})
}
