// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rtsp, TimeSpan } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

const renderTimeSource = Component.renderProp(
  (p: Omit<Select.SingleSimpleProps<rtsp.TimeSource>, "children" | "resourceName">) => (
    <Select.Simple<rtsp.TimeSource> {...p} resourceName="time source">
      <Select.Item itemKey="arrival">Arrival</Select.Item>
      <Select.Item itemKey="capture">Capture</Select.Item>
    </Select.Simple>
  ),
);

interface OffsetProps {
  value: TimeSpan;
  onChange: (value: TimeSpan) => void;
}

const renderOffset = Component.renderProp(({ value, onChange }: OffsetProps) => (
  <Input.Numeric
    value={new TimeSpan(value).milliseconds}
    onChange={(v) => onChange(TimeSpan.milliseconds(v))}
    endContent="ms"
  />
));

const TIME_SOURCE_HELP =
  "Arrival times each frame when it reaches the Core. Capture uses the clock of " +
  "the source, which NTP or PTP must keep in sync.";

const OFFSET_HELP =
  "Added to the time of each frame. A negative offset removes the delay of the " +
  "encoder and the network.";

/** Sets how a video task at `config` times the frames of a channel that saves data. */
export const VideoTimeFields = (): ReactElement => (
  <Flex.Box x>
    <PForm.Field<rtsp.TimeSource>
      path="config.timeSource"
      label="Time source"
      helpText={TIME_SOURCE_HELP}
    >
      {renderTimeSource}
    </PForm.Field>
    <PForm.Field<TimeSpan>
      path="config.offset"
      label="Offset"
      helpText={OFFSET_HELP}
      grow
    >
      {renderOffset}
    </PForm.Field>
  </Flex.Box>
);
