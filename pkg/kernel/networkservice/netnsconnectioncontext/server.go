package netnsconnectioncontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type netnsConnectionContextServer struct{}

func NewServer() networkservice.NetworkServiceServer {
	return &netnsConnectionContextServer{}
}

func (n *netnsConnectionContextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {

	if err := AddIPs(ctx, request.GetConnection(), metadata.IsClient(n)); err != nil {
		return nil, err
	}
	return next.Server(ctx).Request(ctx, request)
}

func (n *netnsConnectionContextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
