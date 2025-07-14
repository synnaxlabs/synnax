// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel, Input } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { type AxisKey, axisLabel } from "@/lineplot/axis";

export interface SelectMultipleAxesInputItemProps
  extends Omit<Input.ItemProps, "onChange" | "children"> {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key[]) => void;
  value: channel.Key[];
  selectProps?: Partial<Channel.SelectMultipleProps>;
}

const SEARCH_OPTIONS: channel.RetrieveOptions = {
  notDataTypes: ["string", "json", "uuid"],
  internal: false,
  virtual: false,
};

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  value,
  selectProps,
  ...rest
}: SelectMultipleAxesInputItemProps): ReactElement => (
  <Input.Item x label={axisLabel(axis)} {...rest}>
    <Channel.SelectMultiple
      value={value}
      initialParams={SEARCH_OPTIONS}
      onChange={useCallback((v: channel.Key[]) => onChange(axis, v), [onChange, axis])}
      {...selectProps}
      style={{ width: "100%" }}
    />
  </Input.Item>
);

export interface SelectAxisInputItemProps extends Omit<Input.ItemProps, "onChange"> {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key) => void;
  value: channel.Key;
  selectProps?: Partial<Channel.SelectSingleProps>;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  value,
  selectProps,
  ...rest
}: SelectAxisInputItemProps): ReactElement => (
  <Input.Item x label={axisLabel(axis)} {...rest}>
    <Channel.SelectSingle
      onChange={useCallback((v: channel.Key) => onChange(axis, v), [axis, onChange])}
      value={value}
      initialParams={SEARCH_OPTIONS}
      {...selectProps}
      style={{ width: "100%" }}
    />
  </Input.Item>
);
