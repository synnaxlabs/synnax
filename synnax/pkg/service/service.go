// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service

import (
	"io"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/effect"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/slate"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/x/config"
)

type Config struct{}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override() error {

}

// Validate implements config.Config.
func (c Config) Validate() error {

}

type Service struct {
	User          *user.Service
	RBAC          *rbac.Service
	Token         *token.Service
	Authenticator auth.Authenticator
	Range         *ranger.Service
	Workspace     *workspace.Service
	Schematic     *schematic.Service
	LinePlot      *lineplot.Service
	Label         *label.Service
	Log           *log.Service
	Table         *table.Service
	Hardware      *hardware.Service
	Framer        *framer.Service
	Slate         *slate.Service
	Effect        *effect.Service
	close         io.Closer
}
