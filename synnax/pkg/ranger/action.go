// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import "github.com/synnaxlabs/synnax/pkg/access/action"

const (
	SetKVAction        action.Action = "set_key_value"
	GetKVAction        action.Action = "get_key_value"
	DeleteKVAction     action.Action = "delete_key_value"
	SetAliasAction     action.Action = "set_alias"
	ResolveAliasAction action.Action = "resolve_alias"
	ListAliasAction    action.Action = "list_alias"
	DeleteAliasAction  action.Action = "delete_alias"
)
