// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align, Divider, Form, type List } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/hardware/ni/device";
import { CustomScaleForm } from "@/hardware/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/hardware/ni/task/MinMaxValueFields";
import {
  type AccelSensitivityUnits,
  type AIChannelType,
  type ElectricalUnits,
  type ForceUnits,
  type PressureUnits,
  type ShuntResistorLoc,
  type TemperatureUnits,
  type TorqueUnits,
  type Units,
  type VelocitySensitivityUnits,
  type VelocityUnits,
} from "@/hardware/ni/task/types";

interface FormProps {
  prefix: string;
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

const SensitivityField = Form.buildNumericField({
  fieldKey: "sensitivity",
  fieldProps: { label: "Sensitivity" },
});

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

const ThermocoupleTypeField = Form.buildDropdownButtonSelectField({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
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

const UnitsField = Form.buildSelectSingleField<Units, KeyedNamed<Units>>({
  fieldKey: "units",
  fieldProps: { label: "Units" },
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

const CHANNEL_FORMS: Record<AIChannelType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <SensitivityField
        path={prefix}
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
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_bridge: ({ prefix }) => (
    <>
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
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_current: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
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
      <ForceUnitsField path={prefix} />
      <MinMaxValueFields path={prefix} />
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
        />
      </Align.Space>
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
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        <ElectricalUnitsField grow path={prefix} />
      </Align.Space>
      <Align.Space direction="x">
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
      <Align.Space direction="x">
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
          { key: "mVoltsPerNewton", name: "mV/N" },
          { key: "mVoltsPerPound", name: "mV/lb" },
        ],
      },
    });
    return (
      <>
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

  ai_microphone: ({ prefix }) => {
    const UnitsField = Form.buildDropdownButtonSelectField({
      fieldKey: "units",
      fieldProps: { label: "Sound Pressure Units" },
      inputProps: {
        entryRenderKey: "name",
        columns: NAMED_KEY_COLS,
        data: [{ key: "Pascals", name: "Pascals" }],
      },
    });
    return (
      <>
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
        />
      </Align.Space>
      <Form.NumericField
        path={`${prefix}.nominalBridgeResistance`}
        label="Nominal Bridge Resistance"
      />
      <Align.Space direction="x">
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Align.Space>
      {/* electricalVals */}
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_pressure_bridge_two_point_lin: ({ prefix }) => (
    <>
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
      <Align.Space direction="x">
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
          style={{ width: "50%" }}
        />
        <ElectricalUnitsField path={prefix} grow style={{ width: "50%" }} />
      </Align.Space>
      <Align.Space direction="x">
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
      <Align.Space direction="x">
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
        <MinMaxValueFields path={prefix} />
        <Divider.Divider direction="x" padded="bottom" />
        <Align.Space direction="x">
          <TemperatureUnitsField path={prefix} grow />
          <RTDTypeField path={prefix} grow />
        </Align.Space>
        <ResistanceConfigField path={prefix} />
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
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x">
          <StrainConfig path={prefix} grow />
          <StrainUnitsField path={prefix} />
        </Align.Space>
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
          />
        </Align.Space>
        <Align.Space direction="x">
          <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" grow />
          <Form.NumericField
            path={`${prefix}.initialBridgeVoltage`}
            label="Initial Bridge Voltage"
            grow
          />
        </Align.Space>
        <Align.Space direction="x">
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
  ai_temp_builtin: ({ prefix }) => <TemperatureUnitsField path={prefix} />,
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
        <MinMaxValueFields path={prefix} />
        <Align.Space direction="x">
          <TemperatureUnitsField path={prefix} grow />
          <ThermocoupleTypeField path={prefix} grow />
        </Align.Space>
        <Align.Space direction="x">
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
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
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
        />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <Align.Space direction="x">
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
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider direction="x" padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
};

export interface AIChannelFormProps {
  type: AIChannelType;
  prefix: string;
}

export const AIChannelForm = ({ type, prefix }: AIChannelFormProps) => {
  const Form = CHANNEL_FORMS[type];
  return (
    <>
      <Align.Space direction="x">
        <Device.Select path={`${prefix}.device`} />
        <Device.PortField path={prefix} />
      </Align.Space>
      <Divider.Divider direction="x" padded="bottom" />
      <Form prefix={prefix} />
    </>
  );
};
