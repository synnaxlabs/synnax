import { useCallback } from "react";

import { Input, InputItemProps } from "@synnaxlabs/pluto";

import { AxisKey, axisLabel } from "../../types";

import {
  SelectChanel,
  SelectChannelProps,
  SelectMultipleChannels,
  SelectMultipleChannelsProps,
} from "@/components";

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
