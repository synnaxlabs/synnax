// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var validateCommand = validate.NewInclusiveBoundsChecker(
	writer.CommandOpen,
	writer.CommandSetAuthority,
)

// validator checks every request against the writer's key set and channel metadata
// before forwarding it to the distribution-layer writer, failing the pipeline with a
// validation error on the first invalid request.
type validator struct {
	confluence.LinearTransform[Request, Request]
	// channels maps each channel key the writer was opened with to its metadata.
	channels map[channel.Key]channel.Channel
}

func newValidator(channels map[channel.Key]channel.Channel) *validator {
	v := &validator{channels: channels}
	v.Transform = v.transform
	return v
}

func (v *validator) transform(
	_ context.Context,
	req Request,
) (Request, bool, error) {
	if err := v.validate(req); err != nil {
		return req, false, err
	}
	return req, true, nil
}

func (v *validator) validate(req Request) error {
	if err := validateCommand(req.Command); err != nil {
		return err
	}
	if req.Command != writer.CommandWrite {
		return nil
	}
	for rawI, k := range req.Frame.RawKeys() {
		if req.Frame.ShouldExcludeRaw(rawI) {
			continue
		}
		ch, ok := v.channels[k]
		if !ok {
			return errors.Wrapf(validate.ErrValidation, "invalid key: %s", k)
		}
		s := req.Frame.RawSeriesAt(rawI)
		if err := s.Validate(); err != nil {
			return errors.Wrapf(err, "channel %s", ch)
		}
		if err := validateSeriesDataType(ch, s); err != nil {
			return err
		}
	}
	return nil
}

func validateSeriesDataType(ch channel.Channel, s telem.Series) error {
	sDt := s.DataType
	cDt := ch.DataType
	isEquivalent := (sDt == telem.Int64T || sDt == telem.TimeStampT) &&
		(cDt == telem.Int64T || cDt == telem.TimeStampT)
	if cDt != sDt && !isEquivalent {
		return errors.Wrapf(
			validate.ErrValidation,
			"channel %s: expected data type %s, got %s",
			ch, cDt, sDt,
		)
	}
	return nil
}
