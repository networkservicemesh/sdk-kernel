package hostdevice

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vppagent/pkg/tools/netnsinode"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	utils "github.com/networkservicemesh/sdk-kernel/pkg/kernel"
)

type kernelHostDeviceServer struct {
	// TODO: keep some sort of link manager wrapper here to make the code testable
}

// NewServer returns NetworkServiceServer chain elements supporting the kernel Mechanism using host devices e.g. SRIOV
func NewServer() networkservice.NetworkServiceServer {
	// TODO: set any additional details
	return &kernelHostDeviceServer{}
}

// Request move host device link into the network namespace
func (dev *kernelHostDeviceServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	mechanism := request.GetConnection().GetMechanism()
	kernelMechanism := kernel.ToMechanism(mechanism)

	pciAddress := kernelMechanism.GetPCIAddress()
	if pciAddress == "" {
		return nil, errors.New("PCI address missing in the connection parameters")
	}

	// TODO: replace the fs pkg
	netnsFileName, err := netnsinode.LinuxNetNSFileName(kernelMechanism.GetNetNSInode())
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup host device")
	}

	targetNetns, err := netns.GetFromPath(netnsFileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup host device")
	}
	defer targetNetns.Close()

	hostNetns, err := netns.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup host device")
	}
	defer hostNetns.Close()

	// always switch back to the host namespace at the end of link setup
	defer func() {
		if err = netns.Set(hostNetns); err != nil {
			logrus.Errorf("failed to switch back to host namespace: %v", err)
		}
	}()

	link, err := utils.FindHostDevice(pciAddress, "", hostNetns, targetNetns)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup host device")
	}

	// move link into the pod's network namespace
	err = link.MoveToNetns(targetNetns)
	if err != nil {
		return errors.Wrap(err, "failed to setup host device")
	}

	return next.Server(ctx).Request(ctx, request)
}

func (dev *kernelHostDeviceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO: remove host device link from the network namespace
	return next.Server(ctx).Close(ctx, conn)
}
