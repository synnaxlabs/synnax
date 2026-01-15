// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form, Icon, Input, Select, state } from "@synnaxlabs/pluto";
import { binary, caseconv, deep, type record } from "@synnaxlabs/x";
import { type DialogFilter } from "@tauri-apps/plugin-dialog";
import { type FC, useRef } from "react";
import { z } from "zod";

import { FS } from "@/fs";
import {
  LINEAR_SCALE_TYPE,
  MAP_SCALE_TYPE,
  NO_SCALE_TYPE,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  TABLE_SCALE_TYPE,
  type Units,
  ZERO_SCALES,
} from "@/hardware/ni/task/types";

const SelectCustomScaleTypeField = Form.buildSelectField<
  ScaleType,
  Select.StaticEntry<ScaleType>
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
    resourceName: "scale type",
    data: [
      { key: LINEAR_SCALE_TYPE, name: "Linear", icon: <Icon.Linear /> },
      { key: MAP_SCALE_TYPE, name: "Map", icon: <Icon.Map /> },
      { key: TABLE_SCALE_TYPE, name: "Table", icon: <Icon.Table /> },
      { key: NO_SCALE_TYPE, name: "None", icon: <Icon.None /> },
    ],
  },
});

const UNIT_SYMBOLS: Record<Units, string> = {
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
};

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

const FILTERS: DialogFilter[] = [{ name: "CSV", extensions: ["csv"] }];

export interface CustomScaleFormProps {
  prefix: string;
}

const CustomScaleUnitsFields = ({ prefix }: { prefix: string }) => (
  <Flex.Box x>
    <UnitsField fieldKey="preScaledUnits" label="Prescaled Units" path={prefix} grow />
    <Form.TextField fieldKey="scaledUnits" label="Scaled Units" path={prefix} grow />
  </Flex.Box>
);

const SCALE_FORMS: Record<ScaleType, FC<CustomScaleFormProps>> = {
  [LINEAR_SCALE_TYPE]: ({ prefix }) => (
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
  [MAP_SCALE_TYPE]: ({ prefix }) => (
    <>
      <CustomScaleUnitsFields prefix={prefix} />
      <Flex.Box x>
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
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField fieldKey="scaledMin" label="Scaled Min" path={prefix} grow />
        <Form.NumericField fieldKey="scaledMax" label="Scaled Max" path={prefix} />
      </Flex.Box>
    </>
  ),
  [TABLE_SCALE_TYPE]: ({ prefix }) => {
    const [rawCol, setRawCol] = state.usePersisted<string>("Raw", `${prefix}.rawCol`);
    const [scaledCol, setScaledCol] = state.usePersisted<string>(
      "Scaled",
      `${prefix}.scaledCol`,
    );
    const [colOptions, setColOptions] = state.usePersisted<record.KeyedNamed<string>[]>(
      [],
      `${prefix}.colOptions`,
    );
    const [path, setPath] = state.usePersisted<string>("", `${prefix}.path`);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const tableSchema = caseconv.preserveCase(
      z.record(z.string(), z.array(z.unknown())),
    );
    const preScaledField = Form.useField<number[]>(`${prefix}.preScaledVals`);
    const scaledField = Form.useField<number[]>(`${prefix}.scaledVals`);
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
      value: z.infer<typeof tableSchema>,
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
        <CustomScaleUnitsFields prefix={prefix} />
        <Input.Item label="Table CSV" padHelpText>
          <FS.InputFileContents<typeof tableSchema>
            initialPath={path}
            onChange={handleFileContentsChange}
            filters={FILTERS}
            decoder={binary.CSV_CODEC}
          />
        </Input.Item>
        <Flex.Box x>
          <Input.Item label="Raw Column" padHelpText grow>
            <Select.Static
              resourceName="raw column"
              value={rawCol}
              onChange={handleRawColChange}
              data={colOptions}
            />
          </Input.Item>
          <Input.Item label="Scaled Column" padHelpText grow>
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
  [NO_SCALE_TYPE]: () => null,
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
