// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact

import (
	"encoding/json/v2"

	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/x/errors"
)

const planChangedType = "sy.impact.plan_changed"

func encode(_ *errors.Codec, err error) (errors.Payload, bool) {
	changed, ok := errors.AsType[impact.PlanChangedError](err)
	if !ok {
		return errors.Payload{}, false
	}
	data, mErr := json.Marshal(changed.Plan)
	if mErr != nil {
		panic(errors.Wrap(mErr, "encode a changed plan"))
	}
	return errors.Payload{Type: planChangedType, Data: string(data)}, true
}

func decode(_ *errors.Codec, p errors.Payload) (error, bool) {
	if p.Type != planChangedType {
		return nil, false
	}
	var changed impact.PlanChangedError
	if err := json.Unmarshal([]byte(p.Data), &changed.Plan); err != nil {
		return errors.Wrap(err, "decode a changed plan"), true
	}
	return changed, true
}

// ErrorProvider encodes and decodes impact.PlanChangedError, with its plan, for an
// errors.Codec.
func ErrorProvider() errors.Provider {
	return errors.Provider{Encode: encode, Decode: decode}
}
