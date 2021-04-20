package vlan

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type vlanMechanismServer struct{}

// NewServer - creates a NetworkServiceServer that requests a vlan interface and populates the netns inode
func NewServer() networkservice.NetworkServiceServer {
	return &vlanMechanismServer{}
}

func (m *vlanMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if mechanism := vlan.ToMechanism(request.GetConnection().GetMechanism()); mechanism != nil {
		mechanism.SetNetNSURL((&url.URL{Scheme: "file", Path: netNSFilename}).String())
	}
	return next.Server(ctx).Request(ctx, request)
}

func (m *vlanMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
