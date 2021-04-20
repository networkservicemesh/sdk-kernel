package netnsconnectioncontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"google.golang.org/grpc"
)

type netnsconnectioncontextClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &netnsconnectioncontextClient{}
}
func (n *netnsconnectioncontextClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if err := AddIPs(ctx, conn, metadata.IsClient(n)); err != nil {
		_, _ = n.Close(ctx, conn, opts...)
		return nil, err
	}
	return conn, nil
}

func (n *netnsconnectioncontextClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
