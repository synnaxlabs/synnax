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
	"encoding/json"

	"github.com/synnaxlabs/x/telem"
)

func TranslateToPB[D any](status Status[D]) (*PBStatus, error) {
	encodedDetails, err := json.Marshal(status.Details)
	if err != nil {
		return nil, err
	}
	return &PBStatus{
		Key:         status.Key,
		Name:        status.Name,
		Variant:     string(status.Variant),
		Message:     status.Message,
		Description: status.Description,
		Time:        int64(status.Time),
		Details:     string(encodedDetails),
	}, nil
}

func TranslateFromPB[D any](pbStatus *PBStatus) (Status[D], error) {
	var details D
	if err := json.Unmarshal([]byte(pbStatus.GetDetails()), &details); err != nil {
		return Status[D]{}, err
	}
	return Status[D]{
		Key:         pbStatus.GetKey(),
		Name:        pbStatus.GetName(),
		Variant:     Variant(pbStatus.GetVariant()),
		Message:     pbStatus.GetMessage(),
		Description: pbStatus.GetDescription(),
		Time:        telem.TimeStamp(pbStatus.GetTime()),
		Details:     details,
	}, nil
}
