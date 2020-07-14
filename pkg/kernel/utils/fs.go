// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package utils contains utility methods for Kernel mechanism machinery
package utils

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"unicode"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netns"
)

// GetNSHandleFromInode return namespace handler from inode
func GetNSHandleFromInode(inode string) (netns.NsHandle, error) {
	/* Parse the string to an integer */
	inodeNum, err := strconv.ParseUint(inode, 10, 64)
	if err != nil {
		return -1, errors.Errorf("failed parsing inode, must be an unsigned int, instead was: %s", inode)
	}
	/* Get filepath from inode */
	nsPath, err := ResolvePodNSByInode(inodeNum)
	if err != nil {
		return -1, errors.Wrapf(err, "failed to find file in /proc/*/ns/net with inode %d", inodeNum)
	}
	/* Get namespace handler from nsPath */
	return netns.GetFromPath(nsPath)
}

// GetInode returns Inode for file
func GetInode(file string) (uint64, error) {
	fileinfo, err := os.Stat(file)
	if err != nil {
		return 0, errors.Wrap(err, "error stat file")
	}
	stat, ok := fileinfo.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("not a stat_t")
	}
	return stat.Ino, nil
}

// ResolvePodNSByInode Traverse /proc/<pid>/<suffix> files,
// compare their inodes with inode parameter and returns file if inode matches
func ResolvePodNSByInode(inode uint64) (string, error) {
	files, err := ioutil.ReadDir("/proc")
	if err != nil {
		return "", errors.Wrap(err, "can't read /proc directory")
	}

	for _, f := range files {
		name := f.Name()
		if isDigits(name) {
			filename := path.Join("/proc", name, "/ns/net")
			tryInode, err := GetInode(filename)
			if err != nil {
				// Just report into log, do not exit
				logrus.Errorf("Can't find %s Error: %v", filename, err)
				continue
			}
			if tryInode == inode {
				if cmdline, err := GetCmdline(name); err == nil && strings.Contains(cmdline, "pause") {
					return filename, nil
				}
			}
		}
	}

	return "", errors.New("not found")
}

// GetCmdline returns /proc/<pid>/cmdline file content
func GetCmdline(pid string) (string, error) {
	data, err := ioutil.ReadFile(path.Clean(path.Join("/proc/", pid, "cmdline")))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func isDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}
