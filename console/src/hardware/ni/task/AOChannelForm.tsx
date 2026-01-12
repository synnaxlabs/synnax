// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form, Icon, Select } from "@synnaxlabs/pluto";
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
  WAVE_TYPES,
  type WaveType,
} from "@/hardware/ni/task/types";

interface SelectWaveTypeProps extends Omit<Select.ButtonsProps<WaveType>, "keys"> {}

const SelectWaveType = (props: SelectWaveTypeProps) => (
  <Select.Buttons {...props} keys={WAVE_TYPES}>
    <Select.Button itemKey={SINE_WAVE_TYPE}>
      <Icon.Wave.Sine />
      Sine
    </Select.Button>
    <Select.Button itemKey={TRIANGLE_WAVE_TYPE}>
      <Icon.Wave.Triangle />
      Triangle
    </Select.Button>
    <Select.Button itemKey={SQUARE_WAVE_TYPE}>
      <Icon.Wave.Square />
      Square
    </Select.Button>
    <Select.Button itemKey={SAWTOOTH_WAVE_TYPE}>
      <Icon.Wave.Sawtooth />
      Sawtooth
    </Select.Button>
  </Select.Buttons>
);

interface FormProps {
  path: string;
}

const CHANNEL_FORMS: Record<AOChannelType, FC<FormProps>> = {
  [AO_CURRENT_CHAN_TYPE]: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
  [AO_FUNC_GEN_CHAN_TYPE]: ({ path }) => (
    <Flex.Box y align="center">
      <Flex.Box x grow>
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
      </Flex.Box>
      <Form.Field<WaveType> path={`${path}.waveType`} showLabel={false}>
        {({ value, onChange }) => <SelectWaveType value={value} onChange={onChange} />}
      </Form.Field>
    </Flex.Box>
  ),
  [AO_VOLTAGE_CHAN_TYPE]: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <Divider.Divider x padded="bottom" />
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
      <Divider.Divider x padded="bottom" />
      <Form path={path} />
    </>
  );
};
