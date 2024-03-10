// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { type channel } from "@synnaxlabs/client";
import { Channel } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";

import { type AxisKey, axisLabel } from "@/vis/axis";

export interface SelectMultipleAxesInputItemProps extends Input.ItemProps {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key[]) => void;
  value: channel.Key[];
}

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  value,
  ...props
}: SelectMultipleAxesInputItemProps): ReactElement => (
  <Input.Item direction="x" label={axisLabel(axis)} {...props}>
    <Channel.SelectMultiple
      value={value}
      onChange={useCallback((v) => onChange(axis, v), [onChange, axis])}
    />
  </Input.Item>
);

export interface SelectAxisInputItemProps extends Input.ItemProps {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key) => void;
  value: channel.Key;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  value,
  ...props
}: SelectAxisInputItemProps): ReactElement => (
  <Input.Item direction="x" label={axisLabel(axis)} {...props}>
    <Channel.SelectSingle
      onChange={useCallback((v) => onChange(axis, v), [axis, onChange])}
      value={value}
    />
  </Input.Item>
);
