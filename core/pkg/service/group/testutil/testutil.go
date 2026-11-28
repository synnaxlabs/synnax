// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil holds helpers for specs that open a group service.
package testutil

import (
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// NameRules returns the name rule of each scope that has one, as the service layer
// gives them to the group service.
func NameRules() map[resource.Type]group.NameRule {
	return map[resource.Type]group.NameRule{
		resource.TypeChannel: channel.CheckFolderName,
	}
}
