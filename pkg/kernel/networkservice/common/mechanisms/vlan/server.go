package vlan

import (
	"context"
	"net/url"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vlanMechanismServer struct {
	baseInterface string
	vlanTag       int32
	isOneLeg      bool
}

// NewServer - creates a NetworkServiceServer that requests a vlan interface and populates the netns inode
func NewServer(baseInterface string, vlanID int32, oneLeg bool) networkservice.NetworkServiceServer {
	v := &vlanMechanismServer{
		baseInterface: baseInterface,
		vlanTag:       vlanID,
		isOneLeg:      oneLeg,
	}
	return v
}

func (v *vlanMechanismServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	log.FromContext(ctx).WithField("vlanMechanismServer", "Request").
		WithField("VlanID", v.vlanTag).
		WithField("BaseInterfaceName", v.baseInterface).
		Debugf("request=", request)

	if conn := request.GetConnection(); conn != nil {
		if mechanism := vlan.ToMechanism(conn.GetMechanism()); mechanism != nil {
			mechanism.SetNetNSURL((&url.URL{Scheme: "file", Path: netNSFilename}).String())
			if conn.GetContext() == nil {
				conn.Context = new(networkservice.ConnectionContext)
			}
			if conn.GetContext().GetEthernetContext() == nil {
				conn.GetContext().EthernetContext = new(networkservice.EthernetContext)
			}
			ethernetContext := conn.GetContext().GetEthernetContext()
			ethernetContext.VlanTag = v.vlanTag

			if conn.GetContext().GetExtraContext() == nil {
				request.Connection.Context.ExtraContext = map[string]string{}
			}
			extracontext := conn.GetContext().GetExtraContext()
			extracontext["baseInterface"] = v.baseInterface
			extracontext["isOneLeg"] = strconv.FormatBool(v.isOneLeg)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (v *vlanMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
