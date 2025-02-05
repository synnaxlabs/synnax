// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Align, Button, Divider, Form, Select } from "@synnaxlabs/pluto";
import { type Keyed } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  CustomScaleForm,
  type FormProps,
  MinMaxValueFields,
} from "@/hardware/ni/task/AIChannelForms";
import { type AOChannelType, type WaveType } from "@/hardware/ni/task/types";

interface WaveTypeEntry extends Keyed<WaveType> {
  icon: React.ReactNode;
  tooltip: string;
}

const WAVE_TYPE_DATA: WaveTypeEntry[] = [
  { key: "Sine", icon: <Icon.Wave.Sine />, tooltip: "Sine Wave" },
  { key: "Square", icon: <Icon.Wave.Square />, tooltip: "Square Wave" },
  { key: "Triangle", icon: <Icon.Wave.Triangle />, tooltip: "Triangle Wave" },
  { key: "Sawtooth", icon: <Icon.Wave.Sawtooth />, tooltip: "Sawtooth Wave" },
];

const SelectWaveType = (props: Select.ButtonProps<WaveType, WaveTypeEntry>) => (
  <Select.Button<WaveType, WaveTypeEntry>
    {...props}
    size="large"
    data={WAVE_TYPE_DATA}
    entryRenderKey="icon"
  >
    {(p) => <SelectButtonOption {...p} />}
  </Select.Button>
);

interface ButtonOptionProps extends Select.ButtonOptionProps<WaveType, WaveTypeEntry> {}

const SelectButtonOption = ({
  key,
  onClick,
  selected,
  entry,
  title,
}: ButtonOptionProps) => (
  <Button.Button
    key={key}
    onClick={onClick}
    variant={selected ? "filled" : "outlined"}
    size="large"
    tooltip={`${entry.key} Wave`}
  >
    {title}
  </Button.Button>
);

const PortField = Form.buildNumericField({
  fieldKey: "port",
  fieldProps: { label: "Port" },
});

export const AO_CHANNEL_FORMS: Record<AOChannelType, FC<FormProps>> = {
  ao_current: ({ prefix }) => (
    <>
      <PortField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ao_voltage: ({ prefix }) => (
    <>
      <PortField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ao_func_gen: ({ prefix }) => (
    <>
      <PortField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="y" align="center">
        <Align.Space direction="x" grow>
          <Form.NumericField path={`${prefix}.frequency`} label="Frequency" grow />
          <Form.NumericField path={`${prefix}.amplitude`} label="Amplitude" grow />
          <Form.NumericField path={`${prefix}.offset`} label="Offset" grow />
        </Align.Space>
        <Form.Field<WaveType> path={`${prefix}.waveType`} showLabel={false}>
          {(p) => <SelectWaveType {...p} />}
        </Form.Field>
      </Align.Space>
    </>
  ),
};
