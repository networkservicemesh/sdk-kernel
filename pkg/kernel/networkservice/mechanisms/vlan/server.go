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

package vlan

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	vlanmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vlan"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

type vlanServer struct{}

// NewServer returns a VLAN server chain element
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

func create(ctx context.Context, conn *networkservice.Connection, isClient bool) error {
	if mechanism := vlanmech.ToMechanism(conn.GetMechanism()); mechanism != nil {
		nsFilename := mechanism.GetNetNSURL()
		if nsFilename == "" {
			return nil
		}
		hostIfName := mechanism.GetInterfaceName()
		vlanID, err := getVlanID(conn)
		if err != nil {
			return nil
		}
		baseInterface, ok := getBaseInterface(conn)
		if !ok {
			return nil
		}

		logger := log.FromContext(ctx).WithField("vlan", "create").
			WithField("HostIfName", hostIfName).
			WithField("HostNamespace", nsFilename).
			WithField("VlanID", vlanID).
			WithField("baseInterface", baseInterface).
			WithField("isClient", isClient)
		logger.Debug("request")

		if isClient {
			return nil
		}
		// TODO generate this based on conn id
		tmpName, _ := generateRandomName(7)

		link, err := createLink(tmpName, baseInterface, vlanID)
		if err != nil {
			return err
		}
		logger.Debugf("Temporary link created Name = %s", tmpName)

		var clientNetNS netns.NsHandle
		clientNetNS, err = nshandle.FromURL(nsFilename)
		if err != nil {
			return errors.Wrapf(err, "handle can not get for client namespace %s", nsFilename)
		}
		defer func() { _ = clientNetNS.Close() }()

		err = moveInterfaceToNamespace(link, clientNetNS)
		if err != nil {
			return err
		}
		logger.Debugf("Moved temporary network interface %s into the client's namespace", tmpName)

		var currNetNS netns.NsHandle
		currNetNS, err = nshandle.Current()
		if err != nil {
			return errors.Wrap(err, "handle can not get for current namespace")
		}
		defer func() { _ = currNetNS.Close() }()

		err = renameInterfaceInNamespace(link, hostIfName, currNetNS, clientNetNS)
		if err != nil {
			return err
		}
		logger.Debug("Network interface set in client namespace")
	}
	return nil
}

func getVlanID(conn *networkservice.Connection) (int, error) {
	if ethernetContext := conn.GetContext().GetEthernetContext(); ethernetContext != nil {
		if ethernetContext.VlanTag != 0 {
			return int(ethernetContext.VlanTag), nil
		}
	}
	return 0, errors.New("no vlanID provided")
}
func getBaseInterface(conn *networkservice.Connection) (string, bool) {
	if extraContext := conn.GetContext().GetExtraContext(); extraContext != nil {
		if baseInterface, ok := extraContext["baseInterface"]; ok {
			return baseInterface, true
		}
	}
	return "", false
}

func generateRandomName(size int) (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return hex.EncodeToString(id)[:size], nil
}

func createLink(name, hostInterface string, vlanID int) (netlink.Link, error) {
	base, err := netlink.LinkByName(hostInterface)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get base interface %s", hostInterface)
	}
	newLink := &netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        name,
			ParentIndex: base.Attrs().Index,
		},
		VlanId:       vlanID,
		VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
	}
	if err = netlink.LinkAdd(newLink); err != nil {
		return nil, errors.Wrapf(err, "failed to create vlan interface %s", name)
	}

	if base.Attrs().OperState != netlink.OperUp {
		if err = netlink.LinkSetUp(base); err != nil {
			return nil, errors.Wrapf(err, "failed to set up host interface: %s", hostInterface)
		}
	}

	return newLink, nil
}

func moveInterfaceToNamespace(link netlink.Link, toNetNS netns.NsHandle) error {
	if err := netlink.LinkSetDown(link); err != nil {
		return errors.Errorf("failed to set %s down: %s", link, err)
	}

	if int(toNetNS) < 0 {
		return errors.Errorf("failed to conver ns handle %s to valid file descriptor", toNetNS)
	}

	if err := netlink.LinkSetNsFd(link, int(toNetNS)); err != nil {
		return errors.Wrapf(err, "failed to move net interface to net NS: %s %s", link, toNetNS)
	}

	return nil
}

func renameInterfaceInNamespace(link netlink.Link, newName string, currNetNS, toNetNS netns.NsHandle) error {
	return nshandle.RunIn(currNetNS, toNetNS, func() error {
		if err := netlink.LinkSetName(link, newName); err != nil {
			return errors.Wrapf(err, "failed to rename net interface:%s -> %s", link, newName)
		}
		err := netlink.LinkSetUp(link)
		if err != nil {
			return errors.Errorf("failed to set %s down: %s", link, err)
		}

		return nil
	})
}
