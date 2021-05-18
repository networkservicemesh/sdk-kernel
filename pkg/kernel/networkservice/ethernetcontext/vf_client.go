package ethernetcontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/networkservice/vfconfig"
)

type vfEthernetClient struct{}

// NewVFClient returns a new VF ethernet context client chain element
func NewVFClient() networkservice.NetworkServiceClient {
	return &vfEthernetClient{}
}

func (i *vfEthernetClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if vfConfig := vfconfig.Config(ctx); vfConfig != nil {
		err := setupEthernetConfig(vfConfig, conn, true)
		if err != nil {
			return nil, err
		}
	}
	return conn, nil
}

func (i *vfEthernetClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
