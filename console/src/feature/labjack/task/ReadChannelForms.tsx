// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Select } from "@synnaxlabs/lyra/select";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { AI_PORT_TYPE, type Model, PORTS } from "@/feature/labjack/device/types";
import {
  AIR_CJC_SOURCE,
  DEVICE_CJC_SOURCE,
  type ReadChannelType,
  type TemperatureUnits,
  type ThermocoupleType,
} from "@/feature/labjack/task/types";

const MaxVoltageField = PForm.buildNumericField({
  fieldKey: "range",
  fieldProps: { label: "Max voltage" },
  inputProps: { endContent: "V" },
});

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
  path: string;
  deviceModel: Model;
}

/** The signal fields for each channel type, or null for a type that has none. */
export const FORMS: Record<ReadChannelType, FC<FormProps> | null> = {
  analog: ({ path }) => <MaxVoltageField path={path} />,
  digital: null,
  thermocouple: ({ path, deviceModel }) => (
    <>
      <ThermocoupleTypeField path={path} />
      <TemperatureUnitsField path={path} />
      <PForm.NumericField fieldKey="negChan" path={path} label="Negative channel" />
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
  ),
};
