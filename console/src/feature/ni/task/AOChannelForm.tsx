// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { type deep } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/ni/device/queries";
import { createTypeFields } from "@/feature/ni/task/createTypeFields";
import { CustomScaleSection } from "@/feature/ni/task/CustomScaleForm";
import {
  DeviceRangeFields,
  type DeviceRangeFieldsProps,
} from "@/feature/ni/task/MinMaxValueFields";
import { PortField } from "@/feature/ni/task/PortField";
import { ANALOG_OUTPUT } from "@/feature/ni/task/ports";
import {
  channelTypeItems,
  SelectAOChannelTypeField,
} from "@/feature/ni/task/SelectAOChannelTypeField";
import {
  AO_CHANNEL_TYPES,
  AO_CURRENT_CHAN_TYPE,
  AO_FUNC_GEN_CHAN_TYPE,
  AO_VOLTAGE_CHAN_TYPE,
  type AOChannel,
  type AOChannelType,
  type WaveType,
} from "@/feature/ni/task/types";
import { allowedTypes } from "@/feature/ni/task/usePort";

interface SelectWaveTypeProps extends Select.ButtonsProps<WaveType> {}

const SelectWaveType = (props: SelectWaveTypeProps) => (
  <Select.Buttons<WaveType> {...props}>
    <Select.Item<WaveType> itemKey="Sine">
      <Icon.Wave.Sine />
      Sine
    </Select.Item>
    <Select.Item<WaveType> itemKey="Triangle">
      <Icon.Wave.Triangle />
      Triangle
    </Select.Item>
    <Select.Item<WaveType> itemKey="Square">
      <Icon.Wave.Square />
      Square
    </Select.Item>
    <Select.Item<WaveType> itemKey="Sawtooth">
      <Icon.Wave.Sawtooth />
      Sawtooth
    </Select.Item>
  </Select.Buttons>
);

type Variant<K extends AOChannelType> = Extract<AOChannel, { type: K }>;

const AORangeFields = (props: Omit<DeviceRangeFieldsProps, "device">) => {
  const device = useFromConfig();
  return <DeviceRangeFields device={device} {...props} />;
};

interface FormProps<K extends AOChannelType = AOChannelType> {
  path: deep.Path<Variant<K>>;
}

const CHANNEL_FORMS: { [K in AOChannelType]: FC<FormProps<K>> } = {
  [AO_CURRENT_CHAN_TYPE]: ({ path }) => (
    <AORangeFields path={path} list="aoCurrentRanges" units="A" />
  ),
  [AO_FUNC_GEN_CHAN_TYPE]: ({ path }) => (
    <>
      <Form.Field path={path.waveType} label="Wave">
        {selectWaveType}
      </Form.Field>
      <Form.NumericField
        path={path.frequency}
        label="Frequency"
        inputProps={HZ_END_CONTENT_INPUT_PROPS}
      />
      <Form.NumericField
        path={path.amplitude}
        label="Amplitude"
        inputProps={V_END_CONTENT_INPUT_PROPS}
      />
      <Form.NumericField
        path={path.offset}
        label="Offset"
        inputProps={V_END_CONTENT_INPUT_PROPS}
      />
    </>
  ),
  [AO_VOLTAGE_CHAN_TYPE]: ({ path }) => (
    <AORangeFields path={path} list="aoVoltageRanges" units="V" />
  ),
};

const HZ_END_CONTENT_INPUT_PROPS = { endContent: "Hz" } as const;

const V_END_CONTENT_INPUT_PROPS = { endContent: "V" } as const;

const selectWaveType = Component.renderProp(SelectWaveType);

export interface AOChannelFormProps {
  type: AOChannelType;
  path: deep.Path<AOChannel>;
  /** The path of the task's channels. */
  channelsPath: deep.Target<AOChannel[]>;
}

const TypeFields = createTypeFields<AOChannel>(CHANNEL_FORMS);

export const AOChannelForm = ({ type, path, channelsPath }: AOChannelFormProps) => {
  const device = useFromConfig();
  const port = Form.useFieldValue(path.port);
  const entry = device?.properties.hardware?.analogOutputs.find((p) => p.port === port);
  const types = allowedTypes(AO_CHANNEL_TYPES, type, entry);
  return (
    <Form.Sections>
      <Form.Section title="Source">
        <PortField
          path={path}
          channelsPath={channelsPath}
          device={device}
          kind={ANALOG_OUTPUT}
          deviceOf={() => device?.key ?? ""}
        />
      </Form.Section>
      <Form.Section title="Signal">
        <SelectAOChannelTypeField
          path={path}
          inputProps={{ children: channelTypeItems(types) }}
        />
        <TypeFields type={type} path={path} />
      </Form.Section>
      <CustomScaleSection path={path} />
    </Form.Sections>
  );
};
