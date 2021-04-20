package vlan

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type vlanServer struct{}

// NewServer returns a VLAN client chain element
func NewServer() networkservice.NetworkServiceServer {
	return &vlanServer{}
}

func (s *vlanServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if err := create(ctx, request.GetConnection(), metadata.IsClient(s)); err != nil {
		return nil, err
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	return conn, err
}

func (s *vlanServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	if _, err := next.Server(ctx).Close(ctx, conn); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}
