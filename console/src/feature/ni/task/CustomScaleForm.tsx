// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Icon, type Select } from "@synnaxlabs/pluto";
import { deep, type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { CoefficientsField } from "@/feature/ni/task/CoefficientsField";
import { selectData } from "@/feature/ni/task/selectData";
import { TableScaleForm } from "@/feature/ni/task/TableScaleForm";
import {
  createScale,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
} from "@/feature/ni/task/types";

const SelectCustomScaleTypeField = Form.buildSelectField<
  ScaleType,
  Select.StaticEntry<ScaleType>
>({
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
    data: [
      { key: "linear", name: "Linear", icon: <Icon.Linear /> },
      { key: "map", name: "Map", icon: <Icon.Map /> },
      { key: "polynomial", name: "Polynomial", icon: <Icon.Function /> },
      { key: "table", name: "Table", icon: <Icon.Table /> },
      { key: "none", name: "None", icon: <Icon.None /> },
    ],
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

const UnitsField = Form.buildSelectField<Units, record.KeyedNamed<Units>>({
  fieldKey: "units",
  fieldProps: { label: "Units", style: { width: "19rem" } },
  inputProps: {
    resourceName: "units",
    allowNone: false,
    data: selectData(UNIT_SYMBOLS),
  },
});

export interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleUnitsFields = ({ prefix }: { prefix: string }) => (
  <Flex.Box x>
    <UnitsField fieldKey="preScaledUnits" label="Prescaled units" path={prefix} grow />
    <Form.TextField fieldKey="scaledUnits" label="Scaled units" path={prefix} grow />
  </Flex.Box>
);

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Flex.Box x>
        <Form.NumericField fieldKey="slope" label="Slope" path={prefix} grow />
        <Form.NumericField
          fieldKey="yIntercept"
          label="Y-Intercept"
          path={prefix}
          grow
        />
      </Flex.Box>
    </>
  ),
  map: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Flex.Box x>
        <Form.NumericField
          fieldKey="preScaledMin"
          label="Pre-scaled min"
          path={prefix}
          grow
        />
        <Form.NumericField
          fieldKey="preScaledMax"
          label="Pre-scaled max"
          path={prefix}
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField fieldKey="scaledMin" label="Scaled min" path={prefix} grow />
        <Form.NumericField fieldKey="scaledMax" label="Scaled max" path={prefix} />
      </Flex.Box>
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

export const CustomScaleForm = ({ prefix }: CustomScaleFormProps) => {
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
