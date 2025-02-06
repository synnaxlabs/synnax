// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Form, Input, type List, Select, state } from "@synnaxlabs/pluto";
import { binary, deep, type KeyedNamed } from "@synnaxlabs/x";
import { type DialogFilter } from "@tauri-apps/plugin-dialog";
import { type FC, useRef } from "react";
import { z } from "zod";

import { FS } from "@/fs";
import {
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type Units,
  ZERO_SCALES,
} from "@/hardware/ni/task/types";

const NAMED_KEY_COLS: List.ColumnSpec<string, KeyedNamed>[] = [
  { key: "name", name: "Name" },
];

const UnitsField = Form.buildSelectSingleField<Units, KeyedNamed<Units>>({
  fieldKey: "units",
  fieldProps: { label: "Units", grow: true },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    allowNone: false,
    data: [
      { key: "Volts", name: "Volts" },
      { key: "Amps", name: "Amps" },
      { key: "DegF", name: "DegF" },
      { key: "DegC", name: "Celsius" },
      { key: "DegR", name: "Rankine" },
      { key: "Kelvins", name: "Kelvins" },
      { key: "Strain", name: "Strain" },
      { key: "Ohms", name: "Ohms" },
      { key: "Hz", name: "Hz" },
      { key: "Seconds", name: "Seconds" },
      { key: "Meters", name: "Meters" },
      { key: "Inches", name: "Inches" },
      { key: "Degrees", name: "Degrees (°)" },
      { key: "Radians", name: "Radians" },
      { key: "g", name: "Gs" },
      { key: "MetersPerSecondSquared", name: "m/s^2" },
      { key: "Newtons", name: "N" },
      { key: "Pounds", name: "lbs" },
      { key: "KilogramForce", name: "kgf" },
      { key: "PoundsPerSquareInch", name: "lbs/in^2" },
      { key: "Bar", name: "Bar" },
      { key: "Pascals", name: "Pa" },
      { key: "VoltsPerVolt", name: "V/V" },
      { key: "mVoltsPerVolt", name: "mV/V" },
      { key: "NewtonMeters", name: "N/M" },
      { key: "InchPounds", name: "in-lbs" },
      { key: "InchOunces", name: "in-oz" },
      { key: "FootPounds", name: "ft-lbs" },
    ],
  },
});

const FILTERS: DialogFilter[] = [{ name: "CSV", extensions: ["csv"] }];

export interface CustomScaleFormProps {
  prefix: string;
}

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  linear: ({ prefix }) => (
    <>
      <Align.Space direction="x" grow>
        <UnitsField fieldKey="preScaledUnits" label="Pre-Scaled Units" path={prefix} />
        <UnitsField fieldKey="scaledUnits" label="Scaled Units" path={prefix} />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.NumericField fieldKey="slope" label="Slope" path={prefix} grow />
        <Form.NumericField
          fieldKey="yIntercept"
          label="Y-Intercept"
          path={prefix}
          grow
        />
      </Align.Space>
    </>
  ),
  map: ({ prefix }) => (
    <>
      <UnitsField fieldKey="preScaledUnits" path={prefix} />
      <Align.Space direction="x" grow>
        <Form.NumericField
          fieldKey="preScaledMin"
          label="Pre-Scaled Min"
          path={prefix}
          grow
        />
        <Form.NumericField
          fieldKey="preScaledMax"
          label="Pre-Scaled Max"
          path={prefix}
        />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.NumericField fieldKey="scaledMin" label="Scaled Min" path={prefix} grow />
        <Form.NumericField fieldKey="scaledMax" label="Scaled Max" path={prefix} />
      </Align.Space>
    </>
  ),
  table: ({ prefix }) => {
    const [rawCol, setRawCol] = state.usePersisted<string>("Raw", `${prefix}.rawCol`);
    const [scaledCol, setScaledCol] = state.usePersisted<string>(
      "Scaled",
      `${prefix}.scaledCol`,
    );
    const [colOptions, setColOptions] = state.usePersisted<KeyedNamed<string>[]>(
      [],
      `${prefix}.colOptions`,
    );
    const [path, setPath] = state.usePersisted<string>("", `${prefix}.path`);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const tableSchema = z.record(z.array(z.unknown()));
    const preScaledField = Form.useField<number[]>({ path: `${prefix}.preScaledVals` });
    const scaledField = Form.useField<number[]>({ path: `${prefix}.scaledVals` });
    const currValueRef = useRef<Record<string, unknown[]>>({});

    const updateValue = () => {
      const value = currValueRef.current;
      const preScaledValues = value[rawCol] as number[] | undefined;
      const scaledValues = value[scaledCol] as number[] | undefined;
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

    const handleFileContentsChange = (
      value: z.output<typeof tableSchema>,
      path: string,
    ) => {
      setPath(path);
      currValueRef.current = value;
      const keys = Object.keys(value).filter(
        (key) =>
          Array.isArray(value[key]) && value[key].every((v) => isFinite(Number(v))),
      );
      setColOptions(keys.map((key) => ({ key, name: key })));
      if (keys.length > 0) setRawCol(keys[0]);
      if (keys.length > 1) setScaledCol(keys[1]);
      updateValue();
    };

    const handleRawColChange = (value: string) => {
      setRawCol(value);
      updateValue();
    };

    const handleScaledColChange = (value: string) => {
      setScaledCol(value);
      updateValue();
    };

    return (
      <>
        <UnitsField fieldKey="preScaledUnits" path={prefix} />
        <Input.Item label="Table CSV" padHelpText>
          <FS.InputFileContents<typeof tableSchema>
            initialPath={path}
            onChange={handleFileContentsChange}
            filters={FILTERS}
            decoder={binary.CSV_CODEC}
          />
        </Input.Item>
        <Align.Space direction="x" grow>
          <Input.Item label="Raw Column" padHelpText grow>
            <Select.Single
              columns={NAMED_KEY_COLS}
              value={rawCol}
              onChange={handleRawColChange}
              data={colOptions}
            />
          </Input.Item>
          <Input.Item label="Scaled Column" padHelpText grow>
            <Select.Single
              columns={NAMED_KEY_COLS}
              value={scaledCol}
              onChange={handleScaledColChange}
              data={colOptions}
            />
          </Input.Item>
        </Align.Space>
      </>
    );
  },
  none: () => <></>,
};

const SelectCustomScaleTypeField = Form.buildDropdownButtonSelectField<
  ScaleType,
  KeyedNamed<ScaleType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Custom Scaling",
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
    columns: NAMED_KEY_COLS,
    data: [
      { key: "linear", name: "Linear" },
      { key: "map", name: "Map" },
      { key: "table", name: "Table" },
      { key: "none", name: "None" },
    ],
  },
});

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
