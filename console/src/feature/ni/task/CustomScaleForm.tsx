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

import { createTypeFields } from "@/feature/ni/task/createTypeFields";
import {
  createScale,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

const SelectType = Form.buildSelectField<ScaleType>({
  fieldProps: { label: "Custom scaling" },
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
  fieldProps: { label: "Units" },
  inputProps: {
    resourceName: "units",
    allowNone: false,
    children: Task.selectItems(UNIT_SYMBOLS),
  },
});

export interface CustomScaleSectionProps {
  path: deep.Target<{ key: string; customScale?: Scale }>;
}
type Variant<K extends ScaleType> = Extract<Scale, { type: K }>;
interface ScaleProps<K extends ScaleType> {
  path: deep.Path<Variant<K>>;
}

const CustomScaleUnitsFields = ({
  path,
}: {
  path: deep.Path<{ preScaledUnits: Units; scaledUnits: string }>;
}) => (
  <>
    <UnitsField label="Prescaled units" path={path.preScaledUnits} />
    <Form.TextField label="Scaled units" path={path.scaledUnits} />
  </>
);

const SCALE_FORMS: { [K in ScaleType]: FC<ScaleProps<K>> } = {
  linear: ({ path }) => (
    <>
      <CustomScaleUnitsFields path={path} />
      <Form.NumericField label="Slope" path={path.slope} />
      <Form.NumericField label="Y-intercept" path={path.yIntercept} />
    </>
  ),
  map: ({ path }) => (
    <>
      <CustomScaleUnitsFields path={path} />
      <Form.NumericField label="Pre-scaled min" path={path.preScaledMin} />
      <Form.NumericField label="Pre-scaled max" path={path.preScaledMax} />
      <Form.NumericField label="Scaled min" path={path.scaledMin} />
      <Form.NumericField label="Scaled max" path={path.scaledMax} />
    </>
  ),
  polynomial: ({ path }) => (
    <>
      <CustomScaleUnitsFields path={path} />
      <Task.CoefficientsField path={path.forwardCoeffs} label="Forward coefficients" />
      <Task.CoefficientsField path={path.reverseCoeffs} label="Reverse coefficients" />
    </>
  ),
  table: ({ path }) => (
    <>
      <CustomScaleUnitsFields path={path} />
      <Task.TableScaleForm path={path} />
    </>
  ),
  none: () => null,
};

const ScaleFields = createTypeFields<Scale>(SCALE_FORMS);

const CustomScaleForm = ({ path }: { path: deep.Path<Scale> }) => {
  const type = Form.useFieldValue(path.type);
  return (
    <>
      <SelectType
        path={path.type}
        onChange={(value, { get, set }) => {
          const prevType = get(path.type).value;
          if (prevType === value) return;
          const next = createScale(value);
          const prev = get(path);
          // Several channels with different scales share no values to keep.
          if (prev.mixed === true) {
            set(path, next);
            return;
          }
          set(path, {
            ...deep.overrideValidItems(next, prev.value, SCALE_SCHEMAS[value]),
            type: next.type,
          } as Scale);
        }}
      />
      <ScaleFields type={type} path={path} />
    </>
  );
};

/** The Scale section, or nothing for a channel type without a custom scale. */
export const CustomScaleSection = ({ path }: CustomScaleSectionProps) => {
  const { customScale } = deep.widen(path);
  const scale = Form.useFieldValue(customScale, { optional: true });
  if (scale == null) return null;
  return (
    <Form.Section title="Scale">
      <CustomScaleForm path={deep.present(customScale)} />
    </Form.Section>
  );
};
