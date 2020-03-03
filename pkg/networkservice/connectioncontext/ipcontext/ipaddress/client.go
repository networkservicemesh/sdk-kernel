package ipaddress

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"google.golang.org/grpc"
)

type setVppIPClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	// TODO
	return nil
}

func (s *setVppIPClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	// TODO
	return conn, err
}

func (s *setVppIPClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	e, err := next.Client(ctx).Close(ctx, conn, opts...)
	// TODO
	return e, err
}
