package vlan

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"google.golang.org/grpc"
)

type vlanClient struct{}

func NewClient() networkservice.NetworkServiceClient {
	return &vlanClient{}
}

func (k *vlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	getMechPref := func(class string) networkservice.Mechanism {
		return networkservice.Mechanism{
			Cls:        class,
			Type:       vlanmech.MECHANISM,
			Parameters: make(map[string]string),
		}
	}
	localMechanism := getMechPref(cls.LOCAL)
	request.MechanismPreferences = append(request.MechanismPreferences, &localMechanism)
	remoteMechanism := getMechPref(cls.REMOTE)
	request.MechanismPreferences = append(request.MechanismPreferences, &remoteMechanism)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	if err := create(ctx, conn, metadata.IsClient(k)); err != nil {
		_, _ = k.Close(ctx, conn, opts...)
		return nil, err
	}
	return conn, nil
}

func (k *vlanClient) Close(
	ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
