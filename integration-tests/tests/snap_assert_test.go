// -*- Mode: Go; indent-tabs-mode: t -*-
// +build !excludeintegration

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package tests

import (
	"path/filepath"

	"gopkg.in/check.v1"

	"github.com/ubuntu-core/snappy/dirs"
	"github.com/ubuntu-core/snappy/integration-tests/testutils/cli"
)

// for cleanup
var dev1AccKeyFiles = filepath.Join(dirs.SnapAssertsDBDir, "asserts-v0/account-key/developer1")

var _ = check.Suite(&snapAssertSuite{})

type snapAssertSuite struct {
	// FIXME: use snapdTestSuite until all tests are moved to
	// assume the snapd/snap command pairing
	snapdTestSuite
}

func (s *snapAssertSuite) TestOK(c *check.C) {
	cli.ExecCommand(c, "sudo", "snap", "assert", "integration-tests/data/dev1.acckey")
	// XXX: forceful cleanup of relevant assertions until we have a better general approach
	defer cli.ExecCommand(c, "sudo", "rm", "-rf", dev1AccKeyFiles)
}
