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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// video view.
func (CreatePayload) Handle(Video) (Video, error) {
	return Video{}, actions.CoreOnlyError("a video create")
}

// Handle sets the video channel the view shows.
func (p SetChannelPayload) Handle(state Video) (Video, error) {
	state.Channel = p.Channel
	return state, nil
}

// Handle sets the range the view plays back, or clears it for live video.
func (p SetPlaybackRangePayload) Handle(state Video) (Video, error) {
	state.PlaybackRange = p.PlaybackRange
	return state, nil
}

// Handle applies the set to the name of the video view.
func (p SetPayload) Handle(state Video) (Video, error) {
	return edit(state, patch.Set(p).Apply)
}

// Handle applies the remove to the name of the video view.
func (p RemovePayload) Handle(state Video) (Video, error) {
	return edit(state, patch.Remove(p).Apply)
}

// Handle applies the insert to the name of the video view.
func (p InsertPayload) Handle(state Video) (Video, error) {
	return edit(state, patch.Insert(p).Apply)
}

// Handle applies the move to the name of the video view.
func (p MovePayload) Handle(state Video) (Video, error) {
	return edit(state, patch.Move(p).Apply)
}

// document is the part of a video view that path actions edit.
type document struct {
	Name string `json:"name"`
}

// edit applies a path action to a copy of the document of the view. A document that
// gains a member it does not declare, such as a path to the group of the view, is
// rejected.
func edit(state Video, apply func(map[string]any) error) (Video, error) {
	d, err := patch.Edit(document{Name: state.Name}, apply)
	if err != nil {
		return Video{}, err
	}
	state.Name = d.Name
	return state, nil
}
