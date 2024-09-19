// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// package action provides a list of common actions that can be performed in API
// requests to the server. Resource-specific actions are defined in the respective
// packages for each resource.
package action

type Action string

const (
	All      Action = "all"
	Copy     Action = "copy"
	Create   Action = "create"
	Delete   Action = "delete"
	Rename   Action = "rename"
	Retrieve Action = "retrieve"
	SetData  Action = "set_data"
)
