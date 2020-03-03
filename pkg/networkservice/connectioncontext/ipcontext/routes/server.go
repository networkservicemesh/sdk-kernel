package routes

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type setKernelIPRouteServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return nil
}

func (s *setKernelIPRouteServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// TODO: set ip route
	return next.Server(ctx).Request(ctx, request)
}

func (s *setKernelIPRouteServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// TODO: unset ip route
	return next.Server(ctx).Close(ctx, conn)
}
