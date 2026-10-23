// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { deep } from "@synnaxlabs/x";
import { type FC } from "react";

import { CoefficientsField } from "@/feature/ni/task/CoefficientsField";
import { TableScaleForm } from "@/feature/ni/task/TableScaleForm";
import {
  createScale,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

const SelectCustomScaleTypeField = Form.buildSelectField<ScaleType>({
  fieldKey: "type",
  fieldProps: {
    label: "Custom scaling",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>(path).value;
      if (prevType === value) return;
      const next = createScale(value);
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
        <Select.Item itemKey="linear">
          <Icon.Linear />
          Linear
        </Select.Item>
        <Select.Item itemKey="map">
          <Icon.Map />
          Map
        </Select.Item>
        <Select.Item itemKey="polynomial">
          <Icon.Function />
          Polynomial
        </Select.Item>
        <Select.Item itemKey="table">
          <Icon.Table />
          Table
        </Select.Item>
        <Select.Item itemKey="none">
          <Icon.None />
          None
        </Select.Item>
      </>
    ),
  },
});

const UNIT_SYMBOLS = {
  Volts: "V",
  Amps: "A",
  DegF: "°F",
  DegC: "°C",
  DegR: "R",
  Kelvins: "K",
  Strain: "strain",
  Ohms: "Ω",
  Hz: "Hz",
  Seconds: "s",
  Meters: "m",
  Inches: "in",
  Degrees: "°",
  Radians: "rad",
  g: "g",
  MetersPerSecondSquared: "m/s^2",
  Newtons: "N",
  Pounds: "lb",
  KilogramForce: "kgf",
  PoundsPerSquareInch: "psi",
  Bar: "bar",
  Pascals: "Pa",
  VoltsPerVolt: "V/V",
  mVoltsPerVolt: "mV/V",
  NewtonMeters: "N·m",
  InchOunces: "in·oz",
  InchPounds: "in·lb",
  FootPounds: "ft·lb",
} as const satisfies Record<Units, string>;

const UnitsField = Form.buildSelectField<Units>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    allowNone: false,
    children: Task.selectItems(UNIT_SYMBOLS),
  },
});

export interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleUnitsFields = ({ prefix }: { prefix: string }) => (
  <>
    <UnitsField fieldKey="preScaledUnits" label="Prescaled units" path={prefix} />
    <Form.TextField fieldKey="scaledUnits" label="Scaled units" path={prefix} />
  </>
);

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Form.NumericField fieldKey="slope" label="Slope" path={prefix} />
      <Form.NumericField fieldKey="yIntercept" label="Y-intercept" path={prefix} />
    </>
  ),
  map: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Form.NumericField fieldKey="preScaledMin" label="Pre-scaled min" path={prefix} />
      <Form.NumericField fieldKey="preScaledMax" label="Pre-scaled max" path={prefix} />
      <Form.NumericField fieldKey="scaledMin" label="Scaled min" path={prefix} />
      <Form.NumericField fieldKey="scaledMax" label="Scaled max" path={prefix} />
    </>
  ),
  polynomial: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <CoefficientsField
        path={`${prefix}.forwardCoeffs`}
        label="Forward coefficients"
      />
      <CoefficientsField
        path={`${prefix}.reverseCoeffs`}
        label="Reverse coefficients"
      />
    </>
  ),
  table: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <TableScaleForm prefix={prefix} />
    </>
  ),
  none: () => null,
};

const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
  const path = `${prefix}.customScale`;
  const type = Form.useFieldValue<ScaleType>(`${path}.type`);
  const FormComponent = SCALE_FORMS[type];
  return (
    <>
      <SelectCustomScaleTypeField path={path} />
      <FormComponent prefix={path} />
    </>
  );
};

/** The Scale section, or nothing for a channel type without a custom scale. */
export const CustomScaleSection = ({ prefix }: CustomScaleFormProps) => {
  const scale = Form.useFieldValue<Scale>(`${prefix}.customScale`, { optional: true });
  if (scale == null) return null;
  return (
    <Form.Section title="Scale">
      <CustomScaleForm prefix={prefix} />
    </Form.Section>
  );
};
