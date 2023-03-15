// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Input, InputItemProps } from "@synnaxlabs/pluto";

import {
  SelectChanel,
  SelectChannelProps,
  SelectMultipleChannels,
  SelectMultipleChannelsProps,
} from "@/components";
import { AxisKey, axisLabel } from "@/vis/types";

export interface SelectMultipleAxesInputItemProps
  extends Omit<
    InputItemProps<readonly string[], readonly string[], SelectMultipleChannelsProps>,
    "onChange" | "label"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: readonly string[]) => void;
}

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  ...props
}: SelectMultipleAxesInputItemProps): JSX.Element => (
  <Input.Item<readonly string[], readonly string[], SelectMultipleChannelsProps>
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
    InputItemProps<string, string, SelectChannelProps>,
    "onChange" | "label"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: string) => void;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  ...props
}: SelectAxisInputItemProps): JSX.Element => (
  <Input.Item<string, string, SelectChannelProps>
    direction="x"
    label={axisLabel(axis) + ":"}
    onChange={useCallback((v) => onChange(axis, v), [axis, onChange])}
    tagKey="name"
    {...props}
  >
    {SelectChanel}
  </Input.Item>
);
