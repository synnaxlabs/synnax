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

import { Device } from "@/feature/labjack/device";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type ReadChannel,
  type ReadChannelType,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type TemperatureUnits,
  type ThermocoupleType,
} from "@/feature/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldProps: { label: "Max voltage" },
  inputProps: { endContent: "V" },
});

const SelectScaleType = PForm.buildSelectField<ScaleType>({
  fieldProps: { label: "Scale" },
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

interface CustomScaleFormProps {
  path: deep.Path<Scale>;
}

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ path }) => {
    const { offset, slope } = deep.variant(path, "type", "linear");
    return (
      <>
        <PForm.NumericField path={slope} label="Slope" />
        <PForm.NumericField path={offset} label="Offset" />
      </>
    );
  },
  map: ({ path }) => {
    const scale = deep.variant(path, "type", "map");
    return (
      <>
        <PForm.NumericField path={scale.preScaledMin} label="Pre-scaled min" />
        <PForm.NumericField path={scale.preScaledMax} label="Pre-scaled max" />
        <PForm.NumericField path={scale.scaledMin} label="Scaled min" />
        <PForm.NumericField path={scale.scaledMax} label="Scaled max" />
      </>
    );
  },
  none: () => null,
};

export const CustomScaleForm = ({ path }: CustomScaleFormProps) => {
  const scaleType = PForm.useFieldValue(path.type);
  const Form = SCALE_FORMS[scaleType];
  return (
    <>
      <SelectScaleType
        path={path.type}
        onChange={(value, { get, set }) => {
          const prevType = get(path.type).value;
          if (prevType === value) return;
          const next = SCALE_SCHEMAS[value].parse({ type: value });
          const prevParent = get(path).value;
          set(path, {
            ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
            type: next.type,
          } as Scale);
        }}
      />
      <Form path={path} />
    </>
  );
};

const ThermocoupleTypeField = PForm.buildSelectField<ThermocoupleType>({
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
  model: Device.Model;
}

const SelectCJCSourceField = ({ model, ...rest }: SelectCJCSourceFieldProps) => {
  const ports: record.KeyedNamed[] = Device.PORTS[model][Device.AI_PORT_TYPE];
  return (
    <Select.Simple<string> allowNone={false} {...rest} resourceName="CJC source">
      <Select.Item itemKey={DEVICE_CJC_SOURCE}>Device</Select.Item>
      <Select.Item itemKey={AIR_CJC_SOURCE}>Air</Select.Item>
      {ports.map(({ key, name }) => (
        <Select.Item key={key} itemKey={key}>
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

interface FormProps {
  path: deep.Path<ReadChannel>;
  deviceModel: Device.Model;
}

/** The signal fields for each channel type, or null for a type that has none. */
export const FORMS: Record<ReadChannelType, FC<FormProps> | null> = {
  analog: ({ path }) => (
    <MaxVoltageField path={deep.variant(path, "type", "analog").range} />
  ),
  digital: null,
  thermocouple: ({ path, deviceModel }) => {
    const channel = deep.variant(path, "type", "thermocouple");
    return (
      <>
        <ThermocoupleTypeField path={channel.thermocoupleType} />
        <TemperatureUnitsField path={channel.units} />
        <PForm.NumericField path={channel.negChan} label="Negative channel" />
        <PForm.Field path={channel.cjcSource} label="CJC source">
          {({ value, onChange, preview }) => (
            <SelectCJCSourceField
              value={value}
              onChange={onChange}
              preview={preview}
              model={deviceModel}
            />
          )}
        </PForm.Field>
        <PForm.NumericField path={channel.cjcSlope} label="CJC slope" />
        <PForm.NumericField path={channel.cjcOffset} label="CJC offset" />
      </>
    );
  },
};
