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
import { Input, Channel, componentRenderProp } from "@synnaxlabs/pluto";

import { AxisKey, axisLabel } from "@/vis/axis";

export interface SelectMultipleAxesInputItemProps
  extends Omit<
    Input.ItemProps<ChannelKey[], ChannelKey[], Channel.SelectMultipleProps>,
    "onChange" | "label" | "data"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: ChannelKey[]) => void;
}

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  ...props
}: SelectMultipleAxesInputItemProps): ReactElement => (
  <Input.Item<ChannelKey[], ChannelKey[], Channel.SelectMultipleProps>
    direction="x"
    label={axisLabel(axis)}
    onChange={useCallback((v) => onChange(axis, v), [onChange, axis])}
    tagKey="name"
    {...props}
  >
    {componentRenderProp(Channel.SelectMultiple)}
  </Input.Item>
);

export interface SelectAxisInputItemProps
  extends Omit<
    Input.ItemProps<ChannelKey, ChannelKey, Channel.SelectSingleProps>,
    "onChange" | "label" | "data"
  > {
  axis: AxisKey;
  onChange: (key: AxisKey, v: ChannelKey) => void;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  ...props
}: SelectAxisInputItemProps): ReactElement => (
  <Input.Item<ChannelKey, ChannelKey, Channel.SelectSingleProps>
    direction="x"
    label={axisLabel(axis)}
    onChange={useCallback((v) => onChange(axis, v), [axis, onChange])}
    tagKey="name"
    {...props}
  >
    {componentRenderProp(Channel.SelectSingle)}
  </Input.Item>
);
