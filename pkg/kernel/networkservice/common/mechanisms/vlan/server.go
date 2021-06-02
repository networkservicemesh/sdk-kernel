// Copyright (c) 2021 Nordix Foundation.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// vlan MechanismServer provides a VLAN client chain element

package vlan

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vlanMechanismServer struct {
	baseInterface string
	vlanTag       int32
}

// NewServer - creates a NetworkServiceServer that requests a vlan interface and populates the netns inode
func NewServer(baseInterface string, vlanID int32) networkservice.NetworkServiceServer {
	v := &vlanMechanismServer{
		baseInterface: baseInterface,
		vlanTag:       vlanID,
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
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (v *vlanMechanismServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
