// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"github.com/synnaxlabs/x/telem"
)

// Variant is a general classification mechanism for statuses.
type Variant string

const (
	InfoVariant     Variant = "info"
	SuccessVariant  Variant = "success"
	ErrorVariant    Variant = "error"
	WarningVariant  Variant = "warning"
	DisabledVariant Variant = "disabled"
	LoadingVariant  Variant = "loading"
)

type Status[D any] struct {
	Key         string          `json:"key" msgpack:"key"`
	Variant     Variant         `json:"variant" msgpack:"variant"`
	Message     string          `json:"message" msgpack:"message"`
	Description string          `json:"description" msgpack:"description"`
	Time        telem.TimeStamp `json:"time" msgpack:"time"`
	Details     D               `json:"details" msgpack:"details"`
}
