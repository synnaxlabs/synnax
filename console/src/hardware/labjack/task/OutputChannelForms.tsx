// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Divider, Form as PForm, Select } from "@synnaxlabs/pluto";
import { deep, type Keyed, type KeyedNamed } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/hardware/labjack/device";
import {
  AI_CHANNEL_TYPE,
  AIR_CJC_SOURCE,
  B_TC_TYPE,
  C_TC_TYPE,
  CELSIUS_UNIT,
  DEVICE_CJC_SOURCE,
  DI_CHANNEL_TYPE,
  E_TC_TYPE,
  FAHRENHEIT_UNIT,
  type InputChannelType,
  J_TC_TYPE,
  K_TC_TYPE,
  KELVIN_UNIT,
  LINEAR_SCALE_TYPE,
  N_TC_TYPE,
  NO_SCALE_TYPE,
  R_TC_TYPE,
  S_TC_TYPE,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  T_TC_TYPE,
  TC_CHANNEL_TYPE,
  type TemperatureUnits,
  type ThermocoupleType,
  ZERO_SCALES,
} from "@/hardware/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldKey: "range",
  fieldProps: { label: "Max Voltage" },
  inputProps: { endContent: "V" },
});

const SelectScaleTypeField = PForm.buildDropdownButtonSelectField<
  ScaleType,
  KeyedNamed<ScaleType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Scale",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_SCALES[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<Scale>(parentPath).value;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
        type: next.type,
      });
    },
  },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: NO_SCALE_TYPE, name: "None" },
      { key: LINEAR_SCALE_TYPE, name: "Linear" },
    ],
  },
});

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  [LINEAR_SCALE_TYPE]: ({ prefix }) => (
    <Align.Space direction="x">
      <PForm.NumericField path={`${prefix}.slope`} label="Slope" grow />
      <PForm.NumericField path={`${prefix}.offset`} label="Offset" grow />
    </Align.Space>
  ),
  [NO_SCALE_TYPE]: () => <></>,
};

interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
  const path = `${prefix}.scale`;
  const scaleType = PForm.useFieldValue<ScaleType>(`${path}.type`);
  const Form = SCALE_FORMS[scaleType];
  return (
    <>
      <SelectScaleTypeField path={path} />
      <Form prefix={path} />
    </>
  );
};

const ThermocoupleTypeField = PForm.buildDropdownButtonSelectField<
  ThermocoupleType,
  KeyedNamed<ThermocoupleType>
>({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    hideColumnHeader: true,
    data: [
      { key: B_TC_TYPE, name: "B" },
      { key: E_TC_TYPE, name: "E" },
      { key: J_TC_TYPE, name: "J" },
      { key: K_TC_TYPE, name: "K" },
      { key: N_TC_TYPE, name: "N" },
      { key: R_TC_TYPE, name: "R" },
      { key: S_TC_TYPE, name: "S" },
      { key: T_TC_TYPE, name: "T" },
      { key: C_TC_TYPE, name: "C" },
    ],
  },
});

const TemperatureUnitsField = PForm.buildDropdownButtonSelectField<
  TemperatureUnits,
  KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: [
      { key: CELSIUS_UNIT, name: "Celsius" },
      { key: FAHRENHEIT_UNIT, name: "Fahrenheit" },
      { key: KELVIN_UNIT, name: "Kelvin" },
    ],
  },
});

interface CJCSourceEntry extends Keyed<string> {}

interface SelectCJCSourceFieldProps extends Select.SingleProps<string, CJCSourceEntry> {
  model: Device.Model;
}

const COLUMNS = [{ key: "key", name: "CJC Source" }];
const DEFAULT_CJC_SOURCE_ENTRIES: CJCSourceEntry[] = [
  { key: DEVICE_CJC_SOURCE },
  { key: AIR_CJC_SOURCE },
];

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const ports: CJCSourceEntry[] = Device.DEVICES[model].ports[Device.AI_PORT_TYPE];
  const data = [...DEFAULT_CJC_SOURCE_ENTRIES, ...ports];
  return (
    <Select.Single<string, CJCSourceEntry>
      data={data}
      columns={COLUMNS}
      allowNone={false}
      entryRenderKey="key"
      {...rest}
    />
  );
};

interface FormProps {
  path: string;
  deviceModel: Device.Model;
}

export const FORMS: Record<InputChannelType, FC<FormProps>> = {
  [AI_CHANNEL_TYPE]: ({ path }) => (
    <>
      <Divider.Divider direction="x" padded="bottom" />
      <MaxVoltageField path={path} />
      <CustomScaleForm prefix={path} />
    </>
  ),
  [DI_CHANNEL_TYPE]: () => <></>,
  [TC_CHANNEL_TYPE]: ({ path, deviceModel }) => (
    <>
      <Divider.Divider direction="x" padded="bottom" />
      <MaxVoltageField path={path} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
        <ThermocoupleTypeField path={path} grow />
        <TemperatureUnitsField path={path} grow />
      </Align.Space>
      <Align.Space direction="x">
        <PForm.NumericField
          fieldKey="posChan"
          path={path}
          label="Positive Channel"
          grow
        />
        <PForm.NumericField
          fieldKey="negChan"
          path={path}
          label="Negative Channel"
          grow
        />
      </Align.Space>
      <Align.Space direction="x">
        <PForm.Field<string>
          path={`${path}.cjcSource`}
          grow
          hideIfNull
          label="CJC Source"
        >
          {(p) => <SelectCJCSourceField {...p} model={deviceModel} />}
        </PForm.Field>
        <PForm.NumericField fieldKey="cjcSlope" path={path} label="CJC Slope" grow />
        <PForm.NumericField fieldKey="cjcOffset" path={path} label="CJC Offset" grow />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={path} />
    </>
  ),
};
