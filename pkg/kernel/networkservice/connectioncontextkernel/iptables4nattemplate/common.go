// Copyright (c) 2022 Xored Software Inc and others.
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

//go:build linux
// +build linux

package iptables4nattemplate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/edwarnicke/exechelper"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk-kernel/pkg/kernel/tools/nshandle"
)

type applyIPTablesKey struct {
}

type iptableManagerImpl struct {
}

// IPTablesManager provides methods for iptables nat rules management
type IPTablesManager interface {
	Get() (string, error)
	Restore(string) error
	Apply([]string) error
}

func (m *iptableManagerImpl) Get() (string, error) {
	cmdStr := "iptables-save"
	buf := bytes.NewBuffer([]byte{})
	err := exechelper.Run(cmdStr,
		exechelper.WithStdout(buf),
		exechelper.WithStderr(buf),
	)
	if err != nil {
		err = errors.Wrapf(err, "%s", buf.String())
		return "", err
	}

	return buf.String(), nil
}

func (m *iptableManagerImpl) Apply(rules []string) error {
	for _, rule := range rules {
		arguments := strings.Split(rule, " ")

		cmdStr := "iptables -t nat"
		buf := bytes.NewBuffer([]byte{})
		err := exechelper.Run(cmdStr,
			exechelper.WithArgs(arguments...),
			exechelper.WithStdout(buf),
			exechelper.WithStderr(buf),
		)
		if err != nil {
			err = errors.Wrapf(err, "%s", buf.String())
			return err
		}
	}

	return nil
}

func (m *iptableManagerImpl) writeTmpRule(rules string) (string, error) {
	fo, err := os.CreateTemp("/tmp", "rules-*")
	if err != nil {
		return "", err
	}

	defer func() { _ = fo.Close() }()
	_, err = fo.WriteString(rules)
	if err != nil {
		return "", err
	}

	return fo.Name(), nil
}

func (m *iptableManagerImpl) Restore(rules string) error {
	// Save rules to a temp file
	tmpFile, err := m.writeTmpRule(rules)
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmpFile) }()

	// Restore rules
	cmdStr := fmt.Sprintf("iptables-restore %s", tmpFile)
	buf := bytes.NewBuffer([]byte{})
	err = exechelper.Run(cmdStr,
		exechelper.WithStdout(buf),
		exechelper.WithStderr(buf),
	)
	if err != nil {
		err = errors.Wrapf(err, "%s", buf.String())
		return err
	}

	return nil
}

func applyIptablesRules(ctx context.Context, conn *networkservice.Connection, c *iptablesClient) error {
	ctxMap := metadata.Map(ctx, metadata.IsClient(c))
	_, rulesWasApplied := ctxMap.Load(applyIPTablesKey{})

	// Check refresh requests
	if rulesWasApplied {
		return nil
	}

	mechanism := kernel.ToMechanism(conn.GetMechanism())
	if mechanism != nil && len(mechanism.GetIPTables4NatTemplate()) != 0 {
		rules, err := mechanism.EvaluateIPTables4NatTemplate(conn)
		if err != nil {
			return err
		}

		currentNsHandler, err := nshandle.Current()
		if err != nil {
			return err
		}
		defer func() { _ = currentNsHandler.Close() }()

		targetHsHandler, err := nshandle.FromURL(mechanism.GetNetNSURL())
		if err != nil {
			return err
		}
		defer func() { _ = targetHsHandler.Close() }()

		err = nshandle.RunIn(currentNsHandler, targetHsHandler, func() error {
			initialRules, iptableErr := c.manager.Get()
			if iptableErr != nil {
				return iptableErr
			}

			ctxMap.Store(applyIPTablesKey{}, initialRules)

			iptableErr = c.manager.Apply(rules)
			if iptableErr != nil {
				return iptableErr
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
