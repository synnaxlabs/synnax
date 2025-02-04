// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Device,
  Divider,
  Form,
  Input,
  type List,
  Select,
  state,
  Synnax,
} from "@synnaxlabs/pluto";
import { binary, deep, type KeyedNamed } from "@synnaxlabs/x";
import { type DialogFilter } from "@tauri-apps/plugin-dialog";
import { type FC, type ReactElement, useRef } from "react";
import { z } from "zod";

import { FS } from "@/fs";
import { Device as NIDevice } from "@/hardware/ni/device";
import { type Properties } from "@/hardware/ni/device/types";
import {
  type AccelSensitivityUnits,
  AI_CHANNEL_SCHEMAS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChan,
  type AIChanType,
  type ElectricalUnits,
  type ForceUnits,
  type PressureUnits,
  type Scale,
  SCALE_SCHEMAS,
  type ScaleType,
  type ShuntResistorLoc,
  type TemperatureUnits,
  type TorqueUnits,
  type Units,
  type VelocitySensitivityUnits,
  type VelocityUnits,
  ZERO_AI_CHANNELS,
  ZERO_SCALES,
} from "@/hardware/ni/task/types";
import { ThermocoupleTypeField } from "@/hardware/task/common/thermocouple";
import { Layout } from "@/layout";

export interface FormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

const NAMED_KEY_COLS: List.ColumnSpec<string, KeyedNamed>[] = [
  { key: "name", name: "Name" },
];

const TerminalConfigField = Form.buildDropdownButtonSelectField<string, KeyedNamed>({
  fieldKey: "terminalConfig",
  fieldProps: { label: "Terminal Configuration" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "RSE", name: "Referenced Single Ended" },
      { key: "NRSE", name: "Non-Referenced Single Ended" },
      { key: "Diff", name: "Differential" },
      { key: "PseudoDiff", name: "Pseudo-Differential" },
      { key: "Cfg_Default", name: "Default" },
    ],
  },
});

const AccelSensitivityUnitsField = Form.buildDropdownButtonSelectField<
  AccelSensitivityUnits,
  KeyedNamed<AccelSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "mVoltsPerG", name: "mV/g" },
      { key: "VoltsPerG", name: "V/g" },
    ],
  },
});

const ExcitSourceField = Form.buildDropdownButtonSelectField<string, KeyedNamed>({
  fieldKey: "excitSource",
  fieldProps: { label: "Excitation Source" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "Internal", name: "Internal" },
      { key: "External", name: "External" },
      { key: "None", name: "None" },
    ],
  },
});

const BridgeConfigField = Form.buildDropdownButtonSelectField<
  string,
  KeyedNamed<string>
>({
  fieldKey: "bridgeConfig",
  fieldProps: { label: "Bridge Configuration" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "FullBridge", name: "Full Bridge" },
      { key: "HalfBridge", name: "Half Bridge" },
      { key: "QuarterBridge", name: "Quarter Bridge" },
    ],
  },
});

const ShuntResistorLocField = Form.buildDropdownButtonSelectField<
  ShuntResistorLoc,
  KeyedNamed<ShuntResistorLoc>
>({
  fieldKey: "shuntResistorLoc",
  fieldProps: { label: "Shunt Resistor Location" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "Default", name: "Default" },
      { key: "Internal", name: "Internal" },
      { key: "External", name: "External" },
    ],
  },
});

const ResistanceConfigField = Form.buildDropdownButtonSelectField<
  string,
  KeyedNamed<string>
>({
  fieldKey: "resistanceConfig",
  fieldProps: { label: "Resistance Configuration" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "2Wire", name: "2-Wire" },
      { key: "3Wire", name: "3-Wire" },
      { key: "4Wire", name: "4-Wire" },
    ],
  },
});

const StrainConfig = Form.buildDropdownButtonSelectField({
  fieldKey: "strainConfig",
  fieldProps: { label: "Strain Configuration" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "FullBridgeI", name: "Full Bridge I" },
      { key: "FullBridgeII", name: "Full Bridge II" },
      { key: "FullBridgeIII", name: "Full Bridge III" },
      { key: "HalfBridgeI", name: "Half Bridge I" },
      { key: "HalfBridgeII", name: "Half Bridge II" },
      { key: "QuarterBridgeI", name: "Quarter Bridge I" },
      { key: "QuarterBridgeII", name: "Quarter Bridge II" },
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

const ForceUnitsField = Form.buildDropdownButtonSelectField<
  ForceUnits,
  KeyedNamed<ForceUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Force Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "Newtons", name: "Newtons" },
      { key: "Pounds", name: "Pounds" },
      { key: "KilogramForce", name: "Kilograms" },
    ],
  },
});

const ElectricalUnitsField = Form.buildDropdownButtonSelectField<
  ElectricalUnits,
  KeyedNamed<ElectricalUnits>
>({
  fieldKey: "electricalUnits",
  fieldProps: { label: "Electrical Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "VoltsPerVolt", name: "V/V" },
      { key: "mVoltsPerVolt", name: "mV/V" },
    ],
  },
});

const PressureUnitsField = Form.buildDropdownButtonSelectField<
  PressureUnits,
  KeyedNamed<PressureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Pressure Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "Pascals", name: "Pascals" },
      { key: "PoundsPerSquareInch", name: "PSI" },
    ],
  },
});

const TemperatureUnitsField = Form.buildDropdownButtonSelectField<
  TemperatureUnits,
  KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature Units" },
  inputProps: {
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: [
      { key: "DegC", name: "Celsius" },
      { key: "DegF", name: "Fahrenheit" },
      { key: "Kelvins", name: "Kelvin" },
      { key: "DegR", name: "Rankine" },
    ],
  },
});

const TorqueUnitsField = Form.buildDropdownButtonSelectField<
  TorqueUnits,
  KeyedNamed<TorqueUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Torque Units" },
  inputProps: {
    columns: NAMED_KEY_COLS,
    entryRenderKey: "name",
    data: [
      { key: "NewtonMeters", name: "Newton Meters" },
      { key: "InchOunces", name: "Inch Ounces" },
      { key: "FootPounds", name: "Foot Pounds" },
    ],
  },
});

const PortField = Form.buildNumericField({
  fieldKey: "port",
  fieldProps: { label: "Port" },
});

export const SelectChannelTypeField = Form.buildSelectSingleField<
  AIChanType,
  KeyedNamed<AIChanType>
>({
  fieldKey: "type",
  fieldProps: {
    label: "Channel Type",
    onChange: (value, { get, set, path }) => {
      const prevType = get<AIChanType>(path).value;
      if (prevType === value) return;
      const next = deep.copy(ZERO_AI_CHANNELS[value]);
      const parentPath = path.slice(0, path.lastIndexOf("."));
      const prevParent = get<AIChan>(parentPath).value;
      let schema = AI_CHANNEL_SCHEMAS[value];
      // @ts-expect-error - schema source type checking
      if ("sourceType" in schema) schema = schema.sourceType() as z.ZodObject<AIChan>;
      set(parentPath, {
        ...deep.overrideValidItems(next, prevParent, schema),
        type: next.type,
      });
    },
  },
  inputProps: {
    hideColumnHeader: true,
    entryRenderKey: "name",
    columns: NAMED_KEY_COLS,
    data: (Object.entries(AI_CHANNEL_TYPE_NAMES) as [AIChanType, string][]).map(
      ([key, name]) => ({ key, name }),
    ),
  },
});

export const UnitsField = Form.buildSelectSingleField<Units, KeyedNamed<Units>>({
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

export const SCALE_FORMS: Record<ScaleType, FC<FormProps>> = {
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

export const SelectCustomScaleTypeField = Form.buildDropdownButtonSelectField<
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

const DevicePortCombo = ({ prefix }: FormProps): ReactElement => (
  <Align.Space direction="x" grow>
    <DeviceField path={prefix} />
    <PortField path={prefix} />
  </Align.Space>
);

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

export interface DeviceFieldProps {
  path: string;
}

export const DeviceField = ({ path }: DeviceFieldProps) => {
  const client = Synnax.use();
  const place = Layout.usePlacer();
  const handleDeviceChange = (v: string) => {
    if (client == null) return;
    client.hardware.devices
      .retrieve<Properties>(v)
      .then(({ configured }) => {
        if (configured) return;
        place(NIDevice.createConfigureLayout(v, {}));
      })
      .catch(console.error);
  };
  return (
    <Form.Field<string>
      path={`${path}.device`}
      label="Device"
      grow
      onChange={handleDeviceChange}
      style={{ width: "100%" }}
    >
      {(p) => (
        <Device.SelectSingle
          allowNone={false}
          grow
          {...p}
          autoSelectOnNone={false}
          searchOptions={{ makes: ["NI"] }}
        />
      )}
    </Form.Field>
  );
};

export const ANALOG_INPUT_FORMS: Record<AIChanType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <TerminalConfigField path={prefix} grow />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_bridge: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <ElectricalUnitsField path={prefix} fieldKey="units" />
      <Align.Space direction="x">
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_current: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <TerminalConfigField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" grow>
        <ShuntResistorLocField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.extShuntResistorVal`}
          label="Shunt Resistance"
          grow
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_bridge_table: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <ForceUnitsField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" size="small">
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
          grow
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" size="small">
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
      <Align.Space direction="x">
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Align.Space>
      {/* electricalVals */}
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_bridge_two_point_lin: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <ForceUnitsField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" grow>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_iepe: ({ prefix }) => {
    const SensitivityUnits = Form.buildDropdownButtonSelectField({
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
        <DevicePortCombo prefix={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <ForceUnitsField path={prefix} inputProps={{ omit: ["KilogramForce"] }} />
        <SensitivityField
          path={prefix}
          inputProps={{
            children: (
              <SensitivityUnits path={prefix} showLabel={false} showHelpText={false} />
            ),
          }}
        />
        <Divider.Divider direction="x" padded="bottom" />
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
        <Divider.Divider direction="x" padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },

  ai_microphone: ({ prefix }) => {
    const UnitsField = Form.buildDropdownButtonSelectField({
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
        <DevicePortCombo prefix={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <TerminalConfigField path={prefix} />
        <UnitsField path={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
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
        <Divider.Divider direction="x" padded="bottom" />
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
        <Divider.Divider direction="x" padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_pressure_bridge_table: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
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
  ),
  ai_pressure_bridge_two_point_lin: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <PressureUnitsField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
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
          style={{ maxWidth: 200 }}
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" grow>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
          style={{ width: "50%" }}
        />
        <ElectricalUnitsField path={prefix} grow style={{ width: "50%" }} />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.NumericField
          path={`${prefix}.firstPhysicalVal`}
          label="Physical Value One"
          grow
          style={{ width: "50%" }}
        />
        <Form.NumericField
          path={`${prefix}.secondPhysicalVal`}
          label="Physical Value Two"
          style={{ width: "50%" }}
          grow
        />
      </Align.Space>
      <Align.Space direction="x" grow>
        <Form.NumericField
          path={`${prefix}.firstElectricalVal`}
          label="Electrical Value One"
          style={{ width: "50%" }}
          grow
        />
        <Form.NumericField
          path={`${prefix}.secondElectricalVal`}
          label="Electrical Value Two"
          style={{ width: "50%" }}
          grow
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_resistance: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <UnitsField path={prefix} />
      <ResistanceConfigField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_rtd: ({ prefix }) => {
    const RTDTypeField = Form.buildDropdownButtonSelectField({
      fieldKey: "rtdType",
      fieldProps: { label: "RTD Type" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        hideColumnHeader: true,
        data: [
          { key: "Pt3750", name: "Pt3750" },
          { key: "Pt3851", name: "Pt3851" },
          { key: "Pt3911", name: "Pt3911" },
          { key: "Pt3916", name: "Pt3916" },
          { key: "Pt3920", name: "Pt3920" },
          { key: "Pt3928", name: "Pt3928" },
        ],
      },
    });
    return (
      <>
        <DevicePortCombo prefix={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <MinMaxValueFields path={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <RTDTypeField path={prefix} grow />
        </Align.Space>
        <ResistanceConfigField path={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
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
        <Form.NumericField path={`${prefix}.r0`} label="R0 Resistance" grow />
      </>
    );
  },
  ai_strain_gauge: ({ prefix }) => {
    const StrainUnitsField = Form.buildDropdownButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Strain Units" },
      inputProps: { data: [{ key: "Strain", name: "Strain" }] },
    });
    return (
      <>
        <DevicePortCombo prefix={prefix} />
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
  ai_temp_builtin: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <TemperatureUnitsField path={prefix} />
    </>
  ),
  ai_thermocouple: ({ prefix }) => {
    const CJCSourceField = Form.buildDropdownButtonSelectField({
      fieldKey: "cjcSource",
      fieldProps: { label: "CJC Source" },
      inputProps: {
        columns: NAMED_KEY_COLS,
        entryRenderKey: "name",
        data: [
          { key: "BuiltIn", name: "Built In" },
          { key: "ConstVal", name: "Constant Value" },
          { key: "Chan", name: "Channel" },
        ],
      },
    });
    const cjcSource = Form.useFieldValue<string>(`${prefix}.cjcSource`, true);
    return (
      <>
        <DevicePortCombo prefix={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x" grow>
          <TemperatureUnitsField path={prefix} grow />
          <ThermocoupleTypeField path={prefix} grow />
        </Align.Space>
        <Align.Space direction="x" grow>
          <CJCSourceField path={prefix} grow />
          {cjcSource === "ConstVal" && (
            <Form.NumericField path={`${prefix}.cjcVal`} label="CJC Value" grow />
          )}
          {cjcSource === "Chan" && (
            <Form.NumericField path={`${prefix}.cjcPort`} label="CJC Port" grow />
          )}
        </Align.Space>
      </>
    );
  },
  ai_torque_bridge_table: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" grow>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
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
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" grow>
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Align.Space>
      {/* electricalVals */}
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_torque_bridge_two_point_lin: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <BridgeConfigField path={prefix} />
      <Form.NumericField
        path={`${prefix}.nominalBridgeResistance`}
        label="Nominal Bridge Resistance"
      />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
        <ExcitSourceField
          path={prefix}
          grow
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x" size="small">
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Align.Space>
      <Align.Space direction="x">
        <Form.NumericField
          grow
          path={`${prefix}.firstPhysicalVal`}
          label="Physical Value One"
        />
        <Form.NumericField
          grow
          path={`${prefix}.secondPhysicalVal`}
          label="Physical Value Two"
        />
      </Align.Space>
      <Align.Space direction="x">
        <Form.NumericField
          grow
          path={`${prefix}.firstElectricalVal`}
          label="Electrical Value One"
        />
        <Form.NumericField
          grow
          path={`${prefix}.secondElectricalVal`}
          label="Electrical Value Two"
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_velocity_iepe: ({ prefix }) => {
    const VelocityUnits = Form.buildDropdownButtonSelectField<
      VelocityUnits,
      KeyedNamed<VelocityUnits>
    >({
      fieldKey: "units",
      fieldProps: { label: "Velocity Units" },
      inputProps: {
        columns: NAMED_KEY_COLS,
        entryRenderKey: "name",
        data: [
          { key: "MetersPerSecond", name: "m/s" },
          { key: "InchesPerSecond", name: "in/s" },
        ],
      },
    });
    const SensitivityUnits = Form.buildDropdownButtonSelectField<
      VelocitySensitivityUnits,
      KeyedNamed<VelocitySensitivityUnits>
    >({
      fieldKey: "sensitivityUnits",
      fieldProps: { label: "Sensitivity Units" },
      inputProps: {
        columns: NAMED_KEY_COLS,
        entryRenderKey: "name",
        data: [
          { key: "MillivoltsPerMillimeterPerSecond", name: "mV/mm/s" },
          { key: "MilliVoltsPerInchPerSecond", name: "mV/in/s" },
        ],
      },
    });
    return (
      <>
        <DevicePortCombo prefix={prefix} />
        <TerminalConfigField path={prefix} />
        <MinMaxValueFields path={prefix} />
        <VelocityUnits path={prefix} />
        <Form.NumericField
          path={`${prefix}.sensitivity`}
          label="Sensitivity"
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
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
          />
        </Align.Space>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage: ({ prefix }) => (
    <>
      <DevicePortCombo prefix={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
};
