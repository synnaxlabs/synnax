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
	"github.com/synnaxlabs/x/zyn"
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


var (
	Variants = []Variant{
		InfoVariant,
		SuccessVariant,
		ErrorVariant,
		WarningVariant,
		DisabledVariant,
		LoadingVariant,
	}
	VariantZ = zyn.Enum(Variants...)
)

// Status is a standardized payload used across Synnax.
type Status[D any] struct {
	// Key is a unique key for the status.
	Key string `json:"key" msgpack:"key"`
	// Variant is the variant of the status.
	Variant Variant `json:"variant" msgpack:"variant"`
	// Message is the message of the status.
	Message string `json:"message" msgpack:"message"`
	// Description is the description of the status.
	Description string `json:"description" msgpack:"description"`
	// Time is the time the status was created.
	Time telem.TimeStamp `json:"time" msgpack:"time"`
	// Details are customizable details for component specific statuses.
	Details D `json:"details" msgpack:"details"`
}



