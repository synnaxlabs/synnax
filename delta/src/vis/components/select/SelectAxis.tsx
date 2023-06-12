// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { ChannelKey } from "@synnaxlabs/client";
import { Input, InputItemProps } from "@synnaxlabs/pluto";

import {
  SelectChanel,
  SelectChannelProps,
  SelectMultipleChannels,
  SelectMultipleChannelsProps,
} from "@/components";
import { AxisKey, axisLabel } from "@/vis/axis";

export interface SelectMultipleAxesInputItemProps
  extends Omit<
    InputItemProps<
      readonly ChannelKey[],
      readonly ChannelKey[],
      SelectMultipleChannelsProps
    >,
    "onChange" | "label"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: readonly ChannelKey[]) => void;
}

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  ...props
}: SelectMultipleAxesInputItemProps): ReactElement => (
  <Input.Item<readonly ChannelKey[], readonly ChannelKey[], SelectMultipleChannelsProps>
    direction="x"
    label={axisLabel(axis) + ":"}
    onChange={useCallback((v) => onChange(axis, v), [onChange, axis])}
    tagKey="name"
    {...props}
  >
    {SelectMultipleChannels}
  </Input.Item>
);

export interface SelectAxisInputItemProps
  extends Omit<
    InputItemProps<ChannelKey, ChannelKey, SelectChannelProps>,
    "onChange" | "label"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: ChannelKey) => void;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  ...props
}: SelectAxisInputItemProps): ReactElement => (
  <Input.Item<ChannelKey, ChannelKey, SelectChannelProps>
    direction="x"
    label={axisLabel(axis) + ":"}
    onChange={useCallback((v) => onChange(axis, v), [axis, onChange])}
    tagKey="name"
    {...props}
  >
    {SelectChanel}
  </Input.Item>
);
