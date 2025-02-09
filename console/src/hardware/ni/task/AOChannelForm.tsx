// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  componentRenderProp,
  Divider,
  Form,
  type Icon as PIcon,
  Select,
} from "@synnaxlabs/pluto";
import { type Keyed } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/hardware/ni/device";
import { CustomScaleForm } from "@/hardware/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/hardware/ni/task/MinMaxValueFields";
import {
  AO_CURRENT_CHAN_TYPE,
  AO_FUNC_GEN_CHAN_TYPE,
  AO_VOLTAGE_CHAN_TYPE,
  type AOChannelType,
  SAWTOOTH_WAVE_TYPE,
  SINE_WAVE_TYPE,
  SQUARE_WAVE_TYPE,
  TRIANGLE_WAVE_TYPE,
  type WaveType,
} from "@/hardware/ni/task/types";

interface WaveTypeEntry extends Keyed<WaveType> {
  icon: PIcon.Element;
}

const WAVE_TYPE_DATA: WaveTypeEntry[] = [
  { key: SINE_WAVE_TYPE, icon: <Icon.Wave.Sine /> },
  { key: SQUARE_WAVE_TYPE, icon: <Icon.Wave.Square /> },
  { key: TRIANGLE_WAVE_TYPE, icon: <Icon.Wave.Triangle /> },
  { key: SAWTOOTH_WAVE_TYPE, icon: <Icon.Wave.Sawtooth /> },
];

interface SelectButtonOptionProps
  extends Select.ButtonOptionProps<WaveType, WaveTypeEntry> {}

const SelectButtonOption = ({
  key,
  onClick,
  selected,
  entry,
  title,
}: SelectButtonOptionProps) => (
  <Button.Button
    key={key}
    onClick={onClick}
    variant={selected ? "filled" : "outlined"}
    size="large"
    tooltip={`${entry.key} Wave`}
    tooltipLocation="top"
  >
    {title}
  </Button.Button>
);

const SelectWaveType = (props: Select.ButtonProps<WaveType, WaveTypeEntry>) => (
  <Select.Button<WaveType, WaveTypeEntry>
    {...props}
    data={WAVE_TYPE_DATA}
    entryRenderKey="icon"
  >
    {(p) => <SelectButtonOption {...p} />}
  </Select.Button>
);

interface FormProps {
  path: string;
}

const CHANNEL_FORMS: Record<AOChannelType, FC<FormProps>> = {
  [AO_CURRENT_CHAN_TYPE]: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
  [AO_FUNC_GEN_CHAN_TYPE]: ({ path }) => (
    <Align.Space direction="y" align="center">
      <Align.Space direction="x" grow>
        <Form.NumericField
          path={`${path}.frequency`}
          label="Frequency"
          inputProps={{ endContent: "Hz" }}
          grow
        />
        <Form.NumericField
          path={`${path}.amplitude`}
          label="Amplitude"
          inputProps={{ endContent: "V" }}
          grow
        />
        <Form.NumericField
          path={`${path}.offset`}
          label="Offset"
          inputProps={{ endContent: "V" }}
          grow
        />
      </Align.Space>
      <Form.Field<WaveType> path={`${path}.waveType`} showLabel={false}>
        {componentRenderProp(SelectWaveType)}
      </Form.Field>
    </Align.Space>
  ),
  [AO_VOLTAGE_CHAN_TYPE]: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
};

export interface AOChannelFormProps {
  type: AOChannelType;
  path: string;
}

export const AOChannelForm = ({ type, path }: AOChannelFormProps) => {
  const Form = CHANNEL_FORMS[type];
  return (
    <>
      <Device.PortField path={path} />
      <Divider.Divider direction="x" padded="bottom" />
      <Form path={path} />
    </>
  );
};
