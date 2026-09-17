// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Form, Icon, Select } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { PortField } from "@/feature/ni/device/PortField";
import { CustomScaleForm } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import { SelectAOChannelTypeField } from "@/feature/ni/task/SelectAOChannelTypeField";
import {
  AO_CURRENT_CHAN_TYPE,
  AO_FUNC_GEN_CHAN_TYPE,
  AO_VOLTAGE_CHAN_TYPE,
  type AOChannelType,
  WAVE_TYPES,
  type WaveType,
} from "@/feature/ni/task/types";

interface SelectWaveTypeProps extends Omit<Select.ButtonsProps<WaveType>, "keys"> {}

const SelectWaveType = (props: SelectWaveTypeProps) => (
  <Select.Buttons<WaveType> {...props} keys={WAVE_TYPES}>
    <Select.Button<WaveType> itemKey="Sine">
      <Icon.Wave.Sine />
      Sine
    </Select.Button>
    <Select.Button<WaveType> itemKey="Triangle">
      <Icon.Wave.Triangle />
      Triangle
    </Select.Button>
    <Select.Button<WaveType> itemKey="Square">
      <Icon.Wave.Square />
      Square
    </Select.Button>
    <Select.Button<WaveType> itemKey="Sawtooth">
      <Icon.Wave.Sawtooth />
      Sawtooth
    </Select.Button>
  </Select.Buttons>
);

interface FormProps {
  path: string;
}

const CHANNEL_FORMS: Record<AOChannelType, FC<FormProps>> = {
  [AO_CURRENT_CHAN_TYPE]: ({ path }) => <MinMaxValueFields path={path} />,
  [AO_FUNC_GEN_CHAN_TYPE]: ({ path }) => (
    <>
      <Form.Field<WaveType> path={`${path}.waveType`} label="Wave">
        {selectWaveType}
      </Form.Field>
      <Form.NumericField
        path={`${path}.frequency`}
        label="Frequency"
        inputProps={HZ_END_CONTENT_INPUT_PROPS}
      />
      <Form.NumericField
        path={`${path}.amplitude`}
        label="Amplitude"
        inputProps={V_END_CONTENT_INPUT_PROPS}
      />
      <Form.NumericField
        path={`${path}.offset`}
        label="Offset"
        inputProps={V_END_CONTENT_INPUT_PROPS}
      />
    </>
  ),
  [AO_VOLTAGE_CHAN_TYPE]: ({ path }) => <MinMaxValueFields path={path} />,
};

const HZ_END_CONTENT_INPUT_PROPS = { endContent: "Hz" } as const;

const V_END_CONTENT_INPUT_PROPS = { endContent: "V" } as const;

const selectWaveType = Component.renderProp(SelectWaveType);

export interface AOChannelFormProps {
  type: AOChannelType;
  path: string;
}

export const AOChannelForm = ({ type, path }: AOChannelFormProps) => {
  const TypeForm = CHANNEL_FORMS[type];
  return (
    <Form.Sections>
      <Form.Section title="Source">
        <PortField path={path} />
      </Form.Section>
      <Form.Section title="Signal">
        <SelectAOChannelTypeField path={path} />
        <TypeForm path={path} />
      </Form.Section>
      {type !== AO_FUNC_GEN_CHAN_TYPE && (
        <Form.Section title="Scale">
          <CustomScaleForm prefix={path} />
        </Form.Section>
      )}
    </Form.Sections>
  );
};
