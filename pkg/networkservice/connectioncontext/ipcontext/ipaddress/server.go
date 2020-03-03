package ipaddress

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type setKernelIPServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	// TODO: what's needed here
	return &setKernelIPServer{}
}

func (s *setKernelIPServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO: set ip on the kernel interface
	return next.Server(ctx).Request(ctx, request)
}

func (s *setKernelIPServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO: delete ip from the kernel interface (if needed)
	return next.Server(ctx).Close(ctx, conn)
}
