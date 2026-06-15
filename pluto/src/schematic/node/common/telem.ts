// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type bounds } from "@synnaxlabs/x";

import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";

export { stringSource, type StringSourceArgs } from "@/vis/value/telem";

export const DEFAULT_THRESHOLD: bounds.Bounds = { lower: 0.9, upper: 1.1 };

/** booleanSource builds the boolean read pipeline for a state channel. */
export const booleanSource = (
  channel: channel.Key = 0,
  threshold: bounds.Bounds = DEFAULT_THRESHOLD,
): telem.BooleanSourceSpec =>
  telem.sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel }),
      threshold: telem.withinBounds({ trueBound: threshold }),
    },
    outlet: "threshold",
  });

/** booleanSink builds the boolean command pipeline for a command channel. */
export const booleanSink = (channel: channel.Key = 0): telem.BooleanSinkSpec =>
  telem.sinkPipeline("boolean", {
    connections: [{ from: "setpoint", to: "setter" }],
    segments: {
      setter: control.setChannelValue({ channel }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  });

/** numberSink builds the numeric command pipeline for a command channel. */
export const numberSink = (channel: channel.Key = 0): telem.NumberSinkSpec =>
  telem.sinkPipeline("number", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel }) },
    inlet: "setter",
  });

/** stringSink builds the string command pipeline for a command channel. */
export const stringSink = (channel: channel.Key = 0): telem.StringSinkSpec =>
  telem.sinkPipeline("string", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel }) },
    inlet: "setter",
  });

/** numberSource builds the numeric read pipeline for a state channel. */
export const numberSource = (channel: channel.Key = 0): telem.NumberSourceSpec =>
  telem.sourcePipeline("number", {
    connections: [],
    segments: { valueStream: telem.streamChannelValue({ channel }) },
    outlet: "valueStream",
  });

export interface ControlChipArgs {
  channel?: channel.Key;
  authority?: number;
}

/** chipStatusSource builds the authority status source for a command channel. */
export const chipStatusSource = (channel: channel.Key = 0): telem.StatusSourceSpec =>
  control.authoritySource({ channel });

/** chipSink builds the control acquisition sink for a command channel. */
export const chipSink = ({
  channel = 0,
  authority = 255,
}: ControlChipArgs): telem.BooleanSinkSpec =>
  control.acquireChannelControl({ channel, authority });
