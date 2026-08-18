// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Icon, Input, Select, state } from "@synnaxlabs/pluto";
import { binary, deep, type record } from "@synnaxlabs/x";
import { type FC } from "react";
import { z } from "zod";

import { CoefficientsField } from "@/feature/ni/task/CoefficientsField";
import {
  createScale,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
} from "@/feature/ni/task/types";
import { FS } from "@/platform/fs";

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

const unitsData = (Object.entries(UNIT_SYMBOLS) as [Units, string][]).map(
  ([key, name]) => ({ key, name }),
);

const UnitsField = Form.buildSelectField<Units, record.KeyedNamed<Units>>({
  fieldKey: "units",
  fieldProps: { label: "Units", style: { width: "19rem" } },
  inputProps: {
    resourceName: "units",
    allowNone: false,
    data: unitsData,
  },
});

const tableSchema = z.record(z.string(), z.array(z.unknown()));

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
  table: ({ prefix }) => {
    const [rawCol, setRawCol] = state.usePersisted<string>("Raw", `${prefix}.rawCol`);
    const [scaledCol, setScaledCol] = state.usePersisted<string>(
      "Scaled",
      `${prefix}.scaledCol`,
    );
    const [colOptions, setColOptions] = state.usePersisted<record.KeyedNamed<string>[]>(
      [],
      `${prefix}.colOptions`,
    );
    const [fileName, setFileName] = state.usePersisted<string>("", `${prefix}.path`);
    // The parsed table persists beside the form state, so a column change after a
    // remount recomputes the values without re-reading the file.
    const [table, setTable] = state.usePersisted<Record<string, unknown[]>>(
      {},
      `${prefix}.table`,
    );
    const preScaledField = Form.useField<number[]>(`${prefix}.preScaledVals`);
    const scaledField = Form.useField<number[]>(`${prefix}.scaledVals`);

    const applyColumns = (
      value: Record<string, unknown[]>,
      raw: string,
      scaled: string,
    ) => {
      const preScaledValues = value[raw] as number[] | undefined;
      const scaledValues = value[scaled] as number[] | undefined;
      const hasScaled = scaledValues != null;
      const hasPreScaled = preScaledValues != null;
      if (hasScaled && hasPreScaled)
        if (preScaledValues.length !== scaledValues.length)
          preScaledField.setStatus({
            variant: "error",
            message: `Pre-scaled ${preScaledValues.length} values and scaled ${scaledValues.length} values must be the same length`,
          });
      if (hasPreScaled) preScaledField.onChange(preScaledValues);
      if (hasScaled) scaledField.onChange(scaledValues);
    };

    const handleFileChange = (value: z.infer<typeof tableSchema>, name: string) => {
      setFileName(name);
      setTable(value);
      const keys = Object.keys(value).filter(
        (key) =>
          Array.isArray(value[key]) && value[key].every((v) => isFinite(Number(v))),
      );
      setColOptions(keys.map((key) => ({ key, name: key })));
      const raw = keys.length > 0 ? keys[0] : rawCol;
      const scaled = keys.length > 1 ? keys[1] : scaledCol;
      if (keys.length > 0) setRawCol(raw);
      if (keys.length > 1) setScaledCol(scaled);
      applyColumns(value, raw, scaled);
    };

    const handleRawColChange = (value: string) => {
      setRawCol(value);
      applyColumns(table, value, scaledCol);
    };

    const handleScaledColChange = (value: string) => {
      setScaledCol(value);
      applyColumns(table, rawCol, value);
    };

    return (
      <>
        <CustomScaleUnitsFields prefix={prefix} />
        <Input.Item label="Table CSV" padHelpText>
          <FS.InputFile<typeof tableSchema>
            value={fileName}
            onChange={handleFileChange}
            extension="csv"
            schema={tableSchema}
            decoder={binary.CSV_CODEC}
          />
        </Input.Item>
        <Flex.Box x>
          <Input.Item label="Raw column" padHelpText grow>
            <Select.Static
              resourceName="raw column"
              value={rawCol}
              onChange={handleRawColChange}
              data={colOptions}
            />
          </Input.Item>
          <Input.Item label="Scaled column" padHelpText grow>
            <Select.Static
              resourceName="scaled column"
              value={scaledCol}
              onChange={handleScaledColChange}
              data={colOptions}
            />
          </Input.Item>
        </Flex.Box>
      </>
    );
  },
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
