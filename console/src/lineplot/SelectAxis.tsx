// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Channel } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { type ReactElement, useCallback } from "react";

import { type AxisKey, axisLabel } from "@/lineplot/axis";

export interface SelectMultipleAxesInputItemProps
  extends Omit<Input.ItemProps, "onChange"> {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key[]) => void;
  value: channel.Key[];
  select?: Channel.SelectMultipleProps;
}

const SEARCH_OPTIONS: channel.RetrieveOptions = {
  notDataTypes: ["string", "json", "uuid"],
  internal: false,
};

export const SelectMultipleAxesInputItem = ({
  axis,
  onChange,
  value,
  select,
  ...props
}: SelectMultipleAxesInputItemProps): ReactElement => (
  <Input.Item direction="x" label={axisLabel(axis)} {...props}>
    <Channel.SelectMultiple
      value={value}
      searchOptions={SEARCH_OPTIONS}
      onChange={useCallback((v: channel.Key[]) => onChange(axis, v), [onChange, axis])}
      {...select}
    />
  </Input.Item>
);

export interface SelectAxisInputItemProps extends Omit<Input.ItemProps, "onChange"> {
  axis: AxisKey;
  onChange: (key: AxisKey, v: channel.Key) => void;
  value: channel.Key;
  select?: Channel.SelectSingleProps;
}

export const SelectAxisInputItem = ({
  axis,
  onChange,
  value,
  select,
  ...props
}: SelectAxisInputItemProps): ReactElement => (
  <Input.Item direction="x" label={axisLabel(axis)} {...props}>
    <Channel.SelectSingle
      onChange={useCallback((v: channel.Key) => onChange(axis, v), [axis, onChange])}
      value={value}
      searchOptions={SEARCH_OPTIONS}
      {...select}
    />
  </Input.Item>
);
