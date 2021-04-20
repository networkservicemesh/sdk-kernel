package vlan

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	netNSFilename = "/proc/thread-self/ns/net"
)

type vlanClient struct {
	interfaceName string
}

func NewClient(options ...Option) networkservice.NetworkServiceClient {
	v := &vlanClient{}
	for _, opt := range options {
		opt(v)
	}
	return v
}

func (v *vlanClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if !v.updateMechanismPreferences(request) {
		request.MechanismPreferences = append(request.GetMechanismPreferences(), &networkservice.Mechanism{
			Cls:  cls.LOCAL,
			Type: vlanmech.MECHANISM,
			Parameters: map[string]string{
				vlanmech.NetNSURL:         (&url.URL{Scheme: "file", Path: netNSFilename}).String(),
				vlanmech.InterfaceNameKey: v.interfaceName,
			},
		})
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

// updateMechanismPreferences returns true if MechanismPreferences has updated
func (v *vlanClient) updateMechanismPreferences(request *networkservice.NetworkServiceRequest) bool {
	var updated = false

	for _, m := range request.GetRequestMechanismPreferences() {
		if m.Type == vlanmech.MECHANISM {
			if m.Parameters == nil {
				m.Parameters = make(map[string]string)
			}
			if m.Parameters[vlanmech.InterfaceNameKey] == "" {
				m.Parameters[vlanmech.InterfaceNameKey] = v.interfaceName
			}
			if m.Parameters[vlanmech.NetNSURL] == "" {
				m.Parameters[vlanmech.NetNSURL] = (&url.URL{Scheme: "file", Path: netNSFilename}).String()
			}
			updated = true
		}
	}

	return updated
}

func (v *vlanClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}

type Option func(v *vlanClient)

// WithInterfaceName sets interface name
func WithInterfaceName(interfaceName string) Option {
	return func(v *vlanClient) {
		v.interfaceName = interfaceName
	}
}
