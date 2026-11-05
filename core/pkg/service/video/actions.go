// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// video view.
func (CreatePayload) Handle(Video) (Video, error) {
	return Video{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches a video create",
	)
}

// Handle replaces the name of the video view.
func (p RenamePayload) Handle(state Video) (Video, error) {
	state.Name = p.Name
	return state, nil
}

// Handle sets the video channel the view shows.
func (p SetChannelPayload) Handle(state Video) (Video, error) {
	state.Channel = p.Channel
	return state, nil
}

// Handle sets the range the view plays back, or clears it for live video.
func (p SetRangePayload) Handle(state Video) (Video, error) {
	state.Range = p.Range
	return state, nil
}
