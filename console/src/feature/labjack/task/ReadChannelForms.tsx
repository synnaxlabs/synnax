// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { deep, type record } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  AI_PORT_TYPE,
  GROUND_CHANNEL,
  type Model,
  negativeChannelOf,
  PORTS,
  T4_MODEL,
  T7_MODEL,
  T8_MODEL,
} from "@/feature/labjack/device/types";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type ReadChannelType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type TemperatureUnits,
  type ThermocoupleType,
} from "@/feature/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldKey: "range",
  fieldProps: { label: "Max voltage" },
  inputProps: { endContent: "V" },
});

const SelectScaleTypeField = PForm.buildSelectField<ScaleType>({
  fieldKey: "type",
  fieldProps: {
    label: "Scale",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = SCALE_SCHEMAS[value].parse({ type: value });
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<Scale>(parentPath).value;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
        type: next.type,
      });
    },
  },
  inputProps: {
    resourceName: "scale type",
    children: (
      <>
        <Select.Item itemKey="none">
          <Icon.None />
          None
        </Select.Item>
        <Select.Item itemKey="linear">
          <Icon.Linear />
          Linear
        </Select.Item>
        <Select.Item itemKey="map">
          <Icon.Map />
          Map
        </Select.Item>
      </>
    ),
  },
});

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <>
      <PForm.NumericField path={`${prefix}.slope`} label="Slope" />
      <PForm.NumericField path={`${prefix}.offset`} label="Offset" />
    </>
  ),
  map: ({ prefix }) => (
    <>
      <PForm.NumericField path={`${prefix}.preScaledMin`} label="Pre-scaled min" />
      <PForm.NumericField path={`${prefix}.preScaledMax`} label="Pre-scaled max" />
      <PForm.NumericField path={`${prefix}.scaledMin`} label="Scaled min" />
      <PForm.NumericField path={`${prefix}.scaledMax`} label="Scaled max" />
    </>
  ),
  none: () => null,
};

interface CustomScaleFormProps {
  prefix: string;
}

export const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
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

const ThermocoupleTypeField = PForm.buildSelectField<ThermocoupleType>({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
    searchName: "type",
    children: (
      <>
        <Select.Item itemKey="B">B</Select.Item>
        <Select.Item itemKey="E">E</Select.Item>
        <Select.Item itemKey="J">J</Select.Item>
        <Select.Item itemKey="K">K</Select.Item>
        <Select.Item itemKey="N">N</Select.Item>
        <Select.Item itemKey="R">R</Select.Item>
        <Select.Item itemKey="S">S</Select.Item>
        <Select.Item itemKey="T">T</Select.Item>
        <Select.Item itemKey="C">C</Select.Item>
      </>
    ),
  },
});

const TemperatureUnitsField = PForm.buildSelectField<TemperatureUnits>({
  fieldKey: "units",
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
    searchName: "unit",
    children: (
      <>
        <Select.Item itemKey="C">Celsius</Select.Item>
        <Select.Item itemKey="F">Fahrenheit</Select.Item>
        <Select.Item itemKey="K">Kelvin</Select.Item>
      </>
    ),
  },
});

interface SelectCJCSourceFieldProps extends Omit<
  Select.SingleSimpleProps<string>,
  "children" | "resourceName"
> {
  model: Model;
}

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const ports: record.KeyedNamed[] = PORTS[model][AI_PORT_TYPE];
  return (
    <Select.Simple<string> allowNone={false} {...rest} resourceName="CJC source">
      {/* On a T8, the device source reads the sensor at the channel's own terminal. */}
      <Select.Item itemKey={DEVICE_CJC_SOURCE}>
        {model === T8_MODEL ? "Terminal" : "Device"}
      </Select.Item>
      {model !== T8_MODEL && <Select.Item itemKey={AIR_CJC_SOURCE}>Air</Select.Item>}
      {ports.map(({ key, name }) => (
        <Select.Item key={key} itemKey={key}>
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

interface FormProps {
  path: string;
  deviceModel: Model;
}

/** The negative input of a T7 channel: ground, or the port's differential pair. */
const NegativeChannelField = ({ path }: Pick<FormProps, "path">) => {
  const port = PForm.useFieldValue<string>(`${path}.port`);
  const pair = negativeChannelOf(port);
  return (
    <PForm.Field<number> path={`${path}.negChan`} label="Negative channel">
      {({ value, onChange, preview }) => (
        <Select.Simple<number>
          allowNone={false}
          value={value}
          onChange={onChange}
          preview={preview}
          resourceName="negative channel"
        >
          <Select.Item itemKey={GROUND_CHANNEL}>Ground</Select.Item>
          {pair != null && <Select.Item itemKey={pair}>{`AIN${pair}`}</Select.Item>}
        </Select.Simple>
      )}
    </PForm.Field>
  );
};

const AnalogForm = ({ path, deviceModel }: FormProps) => (
  <>
    <MaxVoltageField path={path} />
    {deviceModel === T7_MODEL && <NegativeChannelField path={path} />}
  </>
);

const ThermocoupleForm = ({ path, deviceModel }: FormProps) => (
  <>
    <ThermocoupleTypeField path={path} />
    <TemperatureUnitsField path={path} />
    {deviceModel === T7_MODEL && <NegativeChannelField path={path} />}
    <PForm.Field<string> path={`${path}.cjcSource`} label="CJC source">
      {({ value, onChange, preview }) => (
        <SelectCJCSourceField
          value={value}
          onChange={onChange}
          preview={preview}
          model={deviceModel}
        />
      )}
    </PForm.Field>
    <PForm.NumericField fieldKey="cjcSlope" path={path} label="CJC slope" />
    <PForm.NumericField fieldKey="cjcOffset" path={path} label="CJC offset" />
  </>
);

const SIGNAL_FORMS: Record<ReadChannelType, FC<FormProps> | null> = {
  analog: AnalogForm,
  digital: null,
  thermocouple: ThermocoupleForm,
};

/**
 * The signal fields for each model and channel type, or null for a type that has none.
 * The T4 has fixed input ranges and no thermocouple inputs.
 */
export const FORMS: Record<Model, Record<ReadChannelType, FC<FormProps> | null>> = {
  [T4_MODEL]: { analog: null, digital: null, thermocouple: null },
  [T7_MODEL]: SIGNAL_FORMS,
  [T8_MODEL]: SIGNAL_FORMS,
};
