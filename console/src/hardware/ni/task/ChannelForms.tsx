// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Channel,
  Divider,
  Form,
  Input,
  List,
  Select,
  state,
} from "@synnaxlabs/pluto";
import { binary, deep } from "@synnaxlabs/x";
import { FC, ReactElement, useRef } from "react";
import { z } from "zod";

import { FS } from "@/fs";
import {
  AccelerationUnits,
  AccelSensitivityUnits,
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_NAMES,
  AIChan,
  AIChanType,
  ElectricalUnits,
  ForceUnits,
  Scale,
  SCALE_SCHEMAS,
  ScaleType,
  ShuntResistorLoc,
  TorqueUnits,
  Units,
  ZERO_AI_CHANNELS,
  ZERO_SCALES,
} from "@/hardware/ni/task/types";

export interface FormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

export const ChannelField = Form.fieldBuilder(Channel.SelectSingle)({
  fieldKey: "channel",
  inputProps: {
    label: "Synnax Channel",
  },
});

interface NamedKey<K extends string = string> {
  key: K;
  name: string;
}

const NAMED_KEY_COLS: List.ColumnSpec<string, NamedKey>[] = [
  { key: "name", name: "Name" },
];

const TerminalConfigField = Form.buildButtonSelectField<string, NamedKey>({
  fieldKey: "terminalConfig",
  fieldProps: {
    label: "Terminal Configuration",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "RSE",
        name: "Referenced Single Ended",
      },
      {
        key: "NRSE",
        name: "Non-Referenced Single Ended",
      },
      {
        key: "Diff",
        name: "Differential",
      },
      {
        key: "PseudoDiff",
        name: "Pseudo-Differential",
      },
      {
        key: "Cfg_Default",
        name: "Default",
      },
    ],
  },
});

const AccelSensitivityUnitsField = Form.buildButtonSelectField<
  AccelSensitivityUnits,
  NamedKey<AccelSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: {
    label: "Sensitivity Units",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "mVoltsPerG",
        name: "mV/g",
      },
      {
        key: "VoltsPerG",
        name: "V/g",
      },
    ],
  },
});

const ExcitSourceField = Form.buildButtonSelectField<string, NamedKey>({
  fieldKey: "excitSource",
  fieldProps: {
    label: "Excitation Source",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "Internal",
        name: "Internal",
      },
      {
        key: "External",
        name: "External",
      },
      {
        key: "None",
        name: "None",
      },
    ],
  },
});

const AccelerationUnitsField = Form.buildButtonSelectField<
  AccelerationUnits,
  NamedKey<AccelerationUnits>
>({
  fieldKey: "units",
  fieldProps: {
    label: "Acceleration Units",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "g",
        name: "g",
      },
      {
        key: "MetersPerSecondSquared",
        name: "m/s^2",
      },
      {
        key: "InchesPerSecondSquared",
        name: "in/s^2",
      },
    ],
  },
});

const BridgeConfigField = Form.buildButtonSelectField<string, NamedKey<string>>({
  fieldKey: "bridgeConfig",
  fieldProps: {
    label: "Bridge Configuration",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "FullBridge",
        name: "Full Bridge",
      },
      {
        key: "HalfBridge",
        name: "Half Bridge",
      },
      {
        key: "QuarterBridge",
        name: "Quarter Bridge",
      },
    ],
  },
});

const ShuntResistorLocField = Form.buildButtonSelectField<
  ShuntResistorLoc,
  NamedKey<ShuntResistorLoc>
>({
  fieldKey: "shuntResistorLoc",
  fieldProps: {
    label: "Shunt Resistor Location",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "Default",
        name: "Default",
      },
      {
        key: "Internal",
        name: "Internal",
      },
      {
        key: "External",
        name: "External",
      },
    ],
  },
});

const ResistanceConfigField = Form.buildButtonSelectField<string, NamedKey<string>>({
  fieldKey: "resistanceConfig",
  fieldProps: {
    label: "Resistance Configuration",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "2Wire",
        name: "2-Wire",
      },
      {
        key: "3Wire",
        name: "3-Wire",
      },
      {
        key: "4Wire",
        name: "4-Wire",
      },
    ],
  },
});

const StrainConfig = Form.buildButtonSelectField({
  fieldKey: "strainConfig",
  fieldProps: {
    label: "Strain Configuration",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "FullBridgeI",
        name: "Full Bridge I",
      },
      {
        key: "FullBridgeII",
        name: "Full Bridge II",
      },
      {
        key: "FullBridgeIII",
        name: "Full Bridge III",
      },
      {
        key: "HalfBridgeI",
        name: "Half Bridge I",
      },
      {
        key: "HalfBridgeII",
        name: "Half Bridge II",
      },
      {
        key: "QuarterBridgeI",
        name: "Quarter Bridge I",
      },
      {
        key: "QuarterBridgeII",
        name: "Quarter Bridge II",
      },
    ],
  },
});

const MinValueField = Form.buildNumericField({
  fieldKey: "minVal",
  fieldProps: { label: "Minimum Value" },
});
const MaxValueField = Form.buildNumericField({
  fieldKey: "maxVal",
  fieldProps: { label: "Maximum Value" },
});
const SensitivityField = Form.buildNumericField({
  fieldKey: "sensitivity",
  fieldProps: { label: "Sensitivity" },
});

const MinMaxValueFields = ({ path }: { path: string }): ReactElement => (
  <Align.Space direction="x" grow>
    <MinValueField path={path} grow />
    <MaxValueField path={path} grow />
  </Align.Space>
);

const ForceUnitsField = Form.buildButtonSelectField<ForceUnits, NamedKey<ForceUnits>>({
  fieldKey: "units",
  fieldProps: { label: "Force Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "Newtons",
        name: "Newtons",
      },
      {
        key: "Pounds",
        name: "Pounds",
      },
      {
        key: "KilogramForce",
        name: "Kilograms",
      },
    ],
  },
});

const ElectricalUnitsField = Form.buildButtonSelectField<
  ElectricalUnits,
  NamedKey<ElectricalUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Electrical Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "VoltsPerVolt",
        name: "Volts per Volt",
      },
      {
        key: "mVoltsPerVolt",
        name: "mV per Volt",
      },
    ],
  },
});

const PressureUnitsField = Form.buildButtonSelectField({
  fieldKey: "units",
  fieldProps: { label: "Pressure Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "Pascals",
        name: "Pascals",
      },
      {
        key: "PSI",
        name: "PSI",
      },
    ],
  },
});

const TemperatureUnitsField = Form.buildButtonSelectField<string, NamedKey<string>>({
  fieldKey: "units",
  fieldProps: {
    label: "Temperature Units",
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "DegG",
        name: "Celsius",
      },
      {
        key: "DegF",
        name: "Fahrenheit",
      },
      {
        key: "Kelvins",
        name: "Kelvin",
      },
      {
        key: "DegR",
        name: "Rankine",
      },
    ],
  },
});

const TorqueUnitsField = Form.buildButtonSelectField<
  TorqueUnits,
  NamedKey<TorqueUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Torque Units" },
  inputProps: {
    data: [
      {
        key: "NewtonMeters",
        name: "Newton Meters",
      },
      {
        key: "InchOunces",
        name: "Inch Ounces",
      },
      {
        key: "FootPounds",
        name: "Foot Pounds",
      },
    ],
  },
});

const PortField = Form.buildNumericField({
  fieldKey: "port",
  fieldProps: { label: "Port" },
});

export const SelectChannelTypeField = Form.buildSelectSingleField<
  AIChanType,
  NamedKey<AIChanType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "NI Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AIChanType>({ path }).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_AI_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<AIChan>({ path: parentPath }).value;
      set({
        path: parentPath,
        value: {
          ...deep.overrideValidItems(next, prevParent, AI_CHANNEL_SCHEMAS[value]),
          type: next.type,
        },
      });
    },
  },
  inputProps: {
    hideColumnHeader: true,
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: (Object.entries(AI_CHANNEL_TYPE_NAMES) as [AIChanType, string][]).map(
      ([key, name]) => ({
        key,
        name,
      }),
    ),
  },
});

const VoltageUnits = Form.buildButtonSelectField({
  fieldKey: "units",
  fieldProps: { label: "Voltage Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [{ key: "Volts", name: "Volts" }],
  },
});

export const UnitsField = Form.buildSelectSingleField<Units, NamedKey<Units>>({
  fieldKey: "preScaledUnits",
  fieldProps: { label: "Pre-Scaled Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "Volts",
        name: "Volts",
      },
      {
        key: "Amps",
        name: "Amps",
      },
      {
        key: "DegF",
        name: "DegF",
      },
      {
        key: "DegC",
        name: "Celsius",
      },
      {
        key: "DegR",
        name: "Rankine",
      },
      {
        key: "Kelvins",
        name: "Kelvins",
      },
      {
        key: "Strain",
        name: "Strain",
      },
      {
        key: "Ohms",
        name: "Ohms",
      },
      {
        key: "Hz",
        name: "Hz",
      },
      {
        key: "Seconds",
        name: "Seconds",
      },
      {
        key: "Meters",
        name: "Meters",
      },
      {
        key: "Inches",
        name: "Inches",
      },
      {
        key: "Degrees",
        name: "Degrees (°)",
      },
      {
        key: "Radians",
        name: "Radians",
      },
      {
        key: "g",
        name: "Gs",
      },
      {
        key: "MetersPerSecondSquared",
        name: "m/s^2",
      },
      {
        key: "Newtons",
        name: "N",
      },
      {
        key: "Pounds",
        name: "lbs",
      },
      {
        key: "KilogramForce",
        name: "kgf",
      },
      {
        key: "PoundsPerSquareInch",
        name: "lbs/in^2",
      },
      {
        key: "Bar",
        name: "Bar",
      },
      {
        key: "Pascals",
        name: "Pa",
      },
      {
        key: "VoltsPerVolt",
        name: "V/V",
      },
      {
        key: "mVoltsPerVolt",
        name: "mV/V",
      },
      {
        key: "NewtonMeters",
        name: "N/M",
      },
      {
        key: "InchPounds",
        name: "in-lbs",
      },
      {
        key: "InchOunces",
        name: "in-oz",
      },
      {
        key: "FootPounds",
        name: "ft-lbs",
      },
    ],
  },
});

export const SCALE_FORMS: Record<ScaleType, FC<FormProps>> = {
  linear: ({ prefix }) => {
    return (
      <>
        <UnitsField fieldKey="preScaledUnits" label="Pre-Scaled Units" path={prefix} />
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
    );
  },
  map: ({ prefix }) => {
    return (
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
          <Form.NumericField
            fieldKey="scaledMin"
            label="Scaled Min"
            path={prefix}
            grow
          />
          <Form.NumericField fieldKey="scaledMax" label="Scaled Max" path={prefix} />
        </Align.Space>
      </>
    );
  },
  table: ({ prefix }) => {
    const [rawCol, setRawCol] = state.usePersisted<string>("Raw", `${prefix}.rawCol`);
    const [scaledCol, setScaledCol] = state.usePersisted<string>(
      "Scaled",
      `${prefix}.scaledCol`,
    );
    const [colOptions, setColOptions] = state.usePersisted<NamedKey<string>[]>(
      [],
      `${prefix}.colOptions`,
    );
    const [path, setPath] = state.usePersisted<string>("", `${prefix}.path`);
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
      if (hasScaled && hasPreScaled) {
        if (preScaledValues!.length !== scaledValues!.length)
          preScaledField.setStatus({
            variant: "error",
            message: `Pre-scaled ${preScaledValues!.length} values and scaled ${scaledValues!.length} values must be the same length`,
          });
      }
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
            decoder={binary.CSV_ECD}
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

export const SelectCustomScaleTypeField = Form.buildButtonSelectField<
  ScaleType,
  NamedKey<ScaleType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Custom Scaling",
    onChange: (value, { get, set, path }) => {
      const prevType = get<ScaleType>({ path }).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_SCALES[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<Scale>({ path: parentPath }).value;
      set({
        path: parentPath,
        value: {
          ...deep.overrideValidItems(next, prevParent, SCALE_SCHEMAS[value]),
          type: next.type,
        },
      });
    },
  },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      {
        key: "linear",
        name: "Linear",
      },
      {
        key: "map",
        name: "Map",
      },
      {
        key: "table",
        name: "Table",
      },
      {
        key: "none",
        name: "None",
      },
    ],
  },
});

export const CustomScaleForm = ({ prefix }: FormProps): ReactElement => {
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

export const ANALOG_INPUT_FORMS: Record<AIChanType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <ChannelField path={prefix} grow />
      <PortField path={prefix} grow />
      <Align.Space direction="x" grow>
        <TerminalConfigField path={prefix} grow />
        <AccelerationUnitsField
          path={prefix}
          inputProps={{ omit: ["MetersPerSecondSquared", "InchesPerSecondSquared"] }}
          grow
        />
      </Align.Space>
      <MinMaxValueFields path={prefix} />
      <SensitivityField
        path={prefix}
        grow
        inputProps={{
          children: (
            <AccelSensitivityUnitsField
              path={prefix}
              grow
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <Align.Space direction="x" grow>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
      </Align.Space>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_accel_4_wire_dc_voltage: ({ prefix }) => (
    <>
      <ChannelField path={prefix} />
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <SensitivityField
        path={prefix}
        grow
        inputProps={{
          children: (
            <AccelSensitivityUnitsField
              path={prefix}
              grow
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <Align.Space direction="x" grow>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.SwitchField
          path={`${prefix}.useExcitForScaling`}
          label="Use Excitation for Scaling"
        />
      </Align.Space>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_accel_charge: ({ prefix }) => (
    <>
      <ChannelField path={prefix} />
      <PortField path={prefix} />
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <AccelerationUnitsField path={prefix} />
      <SensitivityField
        path={prefix}
        grow
        inputProps={{
          children: (
            <AccelSensitivityUnitsField
              path={prefix}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_bridge: ({ prefix }) => (
    <>
      <ChannelField path={prefix} />
      <PortField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <ElectricalUnitsField path={prefix} />
      <Align.Space direction="x">
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Align.Space>
      <Align.Space direction="x" grow>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
      </Align.Space>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_charge: ({ prefix }) => {
    const Units = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Charge Units" },
      inputProps: {
        data: [
          {
            key: "Coulombs",
            name: "Coulombs",
          },
          {
            key: "PicoCoulombs",
            name: "nC",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Units path={prefix} />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_current: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <ShuntResistorLocField path={prefix} grow />
          <Form.NumericField
            path={`${prefix}.extShuntResistorVal`}
            label="Shunt Resistance"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_current_rms: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <ShuntResistorLocField path={prefix} grow />
          <Form.NumericField
            path={`${prefix}.extShuntResistorVal`}
            label="Shunt Resistance"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  // TODO: Add support for entering coefficients
  ai_force_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <ForceUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        {/* forwardCoeffs */}
        {/* reverseCoeffs */}
        <ElectricalUnitsField path={prefix} />
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_force_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <ForceUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} />
        {/* electricalVals */}
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_force_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <ForceUnitsField path={prefix} />
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="voltageExcitSource"
            label="Voltage Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.voltageExcitVal`}
            label="Voltage Excitation Value"
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <BridgeConfigField path={prefix} grow />
          <Form.NumericField
            path={`${prefix}.nominalBridgeResistance`}
            label="Nominal Bridge Resistance"
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <ForceUnitsField
            path={prefix}
            fieldKey="physicalUnits"
            label="Physical Units"
            grow
          />
          <ElectricalUnitsField grow path={prefix} />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.firstPhysicalVal`}
            label="Physical Value One"
            grow
          />
          <Form.NumericField
            path={`${prefix}.secondPhysicalVal`}
            label="Physical Value Two"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.firstElectricalVal`}
            label="Electrical Value One"
            grow
          />
          <Form.NumericField
            path={`${prefix}.secondElectricalVal`}
            label="Electrical Value Two"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_force_iepe: ({ prefix }) => {
    const SensitivityUnits = Form.buildButtonSelectField({
      fieldKey: "sensitivityUnits",
      fieldProps: { label: "Sensitivity Units" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        data: [
          {
            key: "mVoltsPerNewton",
            name: "mV/N",
          },
          {
            key: "mVoltsPerPound",
            name: "mV/lb",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <ForceUnitsField path={prefix} inputProps={{ omit: ["KilogramForce"] }} />
        <SensitivityField
          path={prefix}
          inputProps={{
            children: (
              <SensitivityUnits path={prefix} showLabel={false} showHelpText={false} />
            ),
          }}
        />
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="currentExcitSource"
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_freq_voltage: ({ prefix }) => {
    const UnitsField = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Frequency Units" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        data: [
          {
            key: "Hz",
            name: "Hertz",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <UnitsField path={prefix} />
        <Align.Space direction="x">
          <Form.NumericField
            path={`${prefix}.thresholdLevel`}
            label="Threshold Level"
            grow
          />
          <Form.NumericField path={`${prefix}.hysteresis`} label="Hysteresis" grow />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_microphone: ({ prefix }) => {
    const UnitsField = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Sound Pressure Units" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        data: [
          {
            key: "Pascals",
            name: "Pascals",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <UnitsField path={prefix} />
        <Align.Space direction="x">
          <Form.NumericField
            path={`${prefix}.micSensitivity`}
            label="Microphone Sensitivity"
            grow
          />
          <Form.NumericField
            path={`${prefix}.maxSndPressLevel`}
            label="Max Sound Pressure Level"
            grow
          />
        </Align.Space>
        <Align.Space direction="x">
          <ExcitSourceField
            path={prefix}
            fieldKey="currentExcitSource"
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_pressure_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <PressureUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        {/* forwardCoeffs */}
        {/* reverseCoeffs */}
        <ElectricalUnitsField path={prefix} />
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_pressure_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <PressureUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} />
        {/* electricalVals */}
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_pressure_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <PressureUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <Align.Space direction="x">
          <ExcitSourceField
            path={prefix}
            fieldKey="voltageExcitSource"
            label="Voltage Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.voltageExcitVal`}
            label="Voltage Excitation Value"
            grow
          />
        </Align.Space>
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        <Align.Space direction="x" grow>
          <PressureUnitsField
            path={prefix}
            fieldKey="physicalUnits"
            label="Physical Units"
            grow
          />
          <ElectricalUnitsField path={prefix} />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.firstPhysicalVal`}
            label="Physical Value One"
            grow
          />
          <Form.NumericField
            path={`${prefix}.secondPhysicalVal`}
            label="Physical Value Two"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.firstElectricalVal`}
            label="Electrical Value One"
            grow
          />
          <Form.NumericField
            path={`${prefix}.secondElectricalVal`}
            label="Electrical Value Two"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_resistance: ({ prefix }) => {
    const UnitsField = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Resistance Units" },
      inputProps: {
        data: [
          {
            key: "Ohms",
            name: "Ohms",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <UnitsField path={prefix} />
        <ResistanceConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_rosette_strain_gage: ({ prefix }) => {
    const TypeField = Form.buildButtonSelectField({
      fieldKey: "rosetteType",
      fieldProps: { label: "Rosette Type" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        data: [
          {
            key: "RectangularRosette",
            name: "Rectangular",
          },
          {
            key: "DeltaRosette",
            name: "Delta",
          },
          {
            key: "TeeRosette",
            name: "Tee",
          },
        ],
      },
    });
    const MeasureTypeField = Form.buildSelectMultiField({
      fieldKey: "rosetteMeasTypes",
      fieldProps: { label: "Measurement Type" },
      inputProps: {
        data: [
          {
            key: "PrincipleStrain1",
            name: "Principle Strain 1",
          },
          {
            key: "PrincipleStrain2",
            name: "Principle Strain 2",
          },
          {
            key: "PrincipleStrainAngle",
            name: "Principle Strain Angle",
          },
          {
            key: "CartesianStrainX",
            name: "Cartesian Strain X",
          },
          {
            key: "CartesianStrainY",
            name: "Cartesian Strain Y",
          },
          {
            key: "CartesianShearStrainXY",
            name: "Cartesian Shear Strain XY",
          },
          {
            key: "MaxShearStrain",
            name: "Max Shear Strain",
          },
          {
            key: "MaxShearStrainAngle",
            name: "Max Shear Strain Angle",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TypeField path={prefix} grow />
          <Form.NumericField
            path={`${prefix}.gageOrientation`}
            label="Gage Orientation"
            grow
          />
        </Align.Space>
        <MeasureTypeField path={prefix} />
        <StrainConfig path={prefix} />
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="voltageExcitSource"
            label="Voltage Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.voltageExcitVal`}
            label="Voltage Excitation Value"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" grow />
          <Form.NumericField
            path={`${prefix}.nominalGageResistance`}
            label="Nominal Gage Resistance"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.poissonRatio`}
            label="Poission's Ratio"
            grow
          />
          <Form.NumericField
            path={`${prefix}.leadWireResistance`}
            label="Lead Wire Resistance"
            grow
          />
        </Align.Space>
      </>
    );
  },
  ai_rtd: ({ prefix }) => {
    const RTDTypeField = Form.buildButtonSelectField({
      fieldKey: "rtdType",
      fieldProps: { label: "RTD Type" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        hideColumnHeader: true,
        data: [
          {
            key: "Pt3750",
            name: "Pt3750",
          },
          {
            key: "Pt3851",
            name: "Pt3851",
          },
          {
            key: "Pt3911",
            name: "Pt3911",
          },
          {
            key: "Pt3916",
            name: "Pt3916",
          },
          {
            key: "Pt3920",
            name: "Pt3920",
          },
          {
            key: "Pt3928",
            name: "Pt3928",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <RTDTypeField path={prefix} grow />
        </Align.Space>
        <ResistanceConfigField path={prefix} />
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="currentExcitSource"
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
            grow
          />
        </Align.Space>
      </>
    );
  },
  ai_strain_gauge: ({ prefix }) => {
    const StrainUnitsField = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Strain Units" },
      inputProps: {
        data: [
          {
            key: "Strain",
            name: "Strain",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <StrainConfig path={prefix} grow />
          <StrainUnitsField path={prefix} />
        </Align.Space>
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="voltageExcitSource"
            label="Voltage Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.voltageExcitVal`}
            label="Voltage Excitation Value"
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" grow />
          <Form.NumericField
            path={`${prefix}.initialBridgeVoltage`}
            label="Initial Bridge Voltage"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField
            path={`${prefix}.nominalGageResistance`}
            label="Nominal Gage Resistance"
            grow
          />

          <Form.NumericField
            path={`${prefix}.poissonRatio`}
            label="Poisson's Ratio"
            grow
          />
          <Form.NumericField
            path={`${prefix}.leadWireResistance`}
            label="Lead Wire Resistance"
            grow
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_temp_builtin: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TemperatureUnitsField path={prefix} />
      </>
    );
  },
  ai_thermocouple: ({ prefix }) => {
    const ThermocoupleTypeField = Form.buildButtonSelectField({
      fieldKey: "thermocoupleType",
      fieldProps: { label: "Thermocouple Type" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        hideColumnHeader: true,
        data: [
          { key: "B", name: "B" },
          { key: "E", name: "E" },
          { key: "J", name: "J" },
          { key: "K", name: "K" },
          { key: "N", name: "N" },
          { key: "R", name: "R" },
          { key: "S", name: "S" },
          { key: "T", name: "T" },
        ],
      },
    });
    const CJCSourceField = Form.buildButtonSelectField({
      fieldKey: "cjcSource",
      fieldProps: { label: "CJC Source" },
      inputProps: {
        data: [
          { key: "BuiltIn", name: "Built In" },
          { key: "ConstVal", name: "Constant Value" },
          { key: "Chan", name: "Channel" },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <ThermocoupleTypeField path={prefix} grow />
        </Align.Space>
        <Align.Space direction="x" grow>
          <CJCSourceField path={prefix} grow />
          <Form.NumericField path={`${prefix}.cjcVal`} label="CJC Value" grow />
        </Align.Space>
      </>
    );
  },
  ai_thermistor_iex: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <ResistanceConfigField path={prefix} grow />
        </Align.Space>
        <Align.Space direction="x">
          <ExcitSourceField
            path={prefix}
            fieldKey="currentExcitSource"
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField path={`${prefix}.a`} label="A" grow />
          <Form.NumericField path={`${prefix}.b`} label="B" grow />
          <Form.NumericField path={`${prefix}.c`} label="C" grow />
        </Align.Space>
      </>
    );
  },
  ai_thermistor_vex: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <ResistanceConfigField path={prefix} grow />
        </Align.Space>
        <Align.Space direction="x" grow>
          <ExcitSourceField
            path={prefix}
            fieldKey="voltageExcitSource"
            label="Voltage Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.voltageExcitVal`}
            label="Voltage Excitation Value"
            grow
          />
        </Align.Space>
        <Align.Space direction="x" grow>
          <Form.NumericField path={`${prefix}.a`} label="A" grow />
          <Form.NumericField path={`${prefix}.b`} label="B" grow />
          <Form.NumericField path={`${prefix}.c`} label="C" grow />
          <Form.NumericField path={`${prefix}.r1`} label="R1" grow />
        </Align.Space>
      </>
    );
  },
  ai_torque_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <TorqueUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        {/* forwardCoeffs */}
        {/* reverseCoeffs */}
        <ElectricalUnitsField path={prefix} />
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_torque_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <TorqueUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} />
        {/* electricalVals */}
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_torque_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <TorqueUnitsField path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <ElectricalUnitsField path={prefix} />
        <Form.NumericField
          path={`${prefix}.firstPhysicalVal`}
          label="Physical Value One"
        />
        <Form.NumericField
          path={`${prefix}.secondPhysicalVal`}
          label="Physical Value Two"
        />
        <Form.NumericField
          path={`${prefix}.firstElectricalVal`}
          label="Electrical Value One"
        />
        <Form.NumericField
          path={`${prefix}.secondElectricalVal`}
          label="Electrical Value Two"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_velocity_iepe: ({ prefix }) => {
    const VelocityUnits = Form.buildButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Velocity Units" },
      inputProps: {
        data: [
          {
            key: "MetersPerSecond",
            name: "m/s",
          },
          {
            key: "InchesPerSecond",
            name: "in/s",
          },
        ],
      },
    });
    const SensitivityUnits = Form.buildButtonSelectField({
      fieldKey: "sensitivityUnits",
      fieldProps: { label: "Sensitivity Units" },
      inputProps: {
        data: [
          {
            key: "MillivoltsPerMillimeterPerSecond",
            name: "mV/mm/s",
          },
          {
            key: "MilliVoltsPerInchPerSecond",
            name: "mV/in/s",
          },
        ],
      },
    });
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <ElectricalUnitsField path={prefix} />
        <VelocityUnits path={prefix} />
        <Form.NumericField path={`${prefix}.sensitivity`} label="Sensitivity" />
        <SensitivityUnits path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage: ({ prefix }) => {
    return (
      <>
        <Align.Space direction="x" grow>
          <ChannelField path={prefix} grow />
          <PortField path={prefix} style={{ width: 100 }} />
        </Align.Space>
        <Divider.Divider direction="x" padded />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Divider.Divider direction="x" padded />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage_rms: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <VoltageUnits path={prefix} />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage_with_excit: ({ prefix }) => {
    return (
      <>
        <ChannelField path={prefix} />
        <PortField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <VoltageUnits path={prefix} />
        <BridgeConfigField path={prefix} />
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.SwitchField
          path={`${prefix}.useExcitForScaling`}
          label="Use Excitation for Scaling"
        />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
};
