package kernel

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk-kernel/pkg/networkservice/mechanisms/kernel/hostdevice"
)

func NewServer() networkservice.NetworkServiceServer {
	// TODO
	return hostdevice.NewServer()
}
