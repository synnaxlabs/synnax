// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ni/task/AIChannelForm.css";

import { Divider, Flex, Form, Icon, type Select as PSelect } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";

import { PortField } from "@/feature/ni/device/PortField";
import { Select } from "@/feature/ni/device/Select";
import { CoefficientsField } from "@/feature/ni/task/CoefficientsField";
import { CustomScaleForm } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import {
  type AccelChargeSensitivityUnits,
  type AccelSensitivityUnits,
  type AccelUnits,
  type AIChannelType,
  type ChargeUnits,
  type CJC,
  type CJCType,
  type ElectricalUnits,
  type ForceUnits,
  type PressureUnits,
  type ShuntResistorLoc,
  type TemperatureUnits,
  type TorqueUnits,
  type VelocitySensitivityUnits,
  type VelocityUnits,
} from "@/feature/ni/task/types";
import { CSS } from "@/platform/css";

interface FormProps {
  prefix: string;
}

const TerminalConfigField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalConfig",
  fieldProps: { label: "Terminal configuration" },
  inputProps: {
    resourceName: "terminal configuration",
    data: [
      { key: "RSE", name: "Referenced single ended" },
      { key: "NRSE", name: "Non-referenced single ended" },
      { key: "Diff", name: "Differential" },
      { key: "PseudoDiff", name: "Pseudo-Differential" },
      { key: "Cfg_Default", name: "Default" },
    ],
  },
});

const AccelUnitsField = Form.buildSelectField<
  AccelUnits,
  record.KeyedNamed<AccelUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Acceleration units" },
  inputProps: {
    resourceName: "acceleration units",
    data: [
      { key: "g", name: "g" },
      { key: "MetersPerSecondSquared", name: "m/s²" },
      { key: "InchesPerSecondSquared", name: "in/s²" },
    ],
  },
});

const AccelSensitivityUnitsField = Form.buildSelectField<
  AccelSensitivityUnits,
  record.KeyedNamed<AccelSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: [
      { key: "mVoltsPerG", name: "mV/g" },
      { key: "VoltsPerG", name: "V/g" },
    ],
  },
});

const AccelChargeSensitivityUnitsField = Form.buildSelectField<
  AccelChargeSensitivityUnits,
  record.KeyedNamed<AccelChargeSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: [
      { key: "PicoCoulombsPerG", name: "pC/g" },
      { key: "PicoCoulombsPerMetersPerSecondSquared", name: "pC/(m/s²)" },
      { key: "PicoCoulombsPerInchesPerSecondSquared", name: "pC/(in/s²)" },
    ],
  },
});

const ExcitSourceField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "excitSource",
  fieldProps: { label: "Excitation source" },
  inputProps: {
    resourceName: "excitation source",
    data: [
      { key: "Internal", name: "Internal" },
      { key: "External", name: "External" },
      { key: "None", name: "None" },
    ],
  },
});

const BridgeConfigField = Form.buildSelectField<string, record.KeyedNamed<string>>({
  fieldKey: "bridgeConfig",
  fieldProps: { label: "Bridge configuration" },
  inputProps: {
    resourceName: "bridge configuration",
    data: [
      { key: "FullBridge", name: "Full bridge" },
      { key: "HalfBridge", name: "Half bridge" },
      { key: "QuarterBridge", name: "Quarter bridge" },
    ],
  },
});

const ShuntResistorLocField = Form.buildSelectField<
  ShuntResistorLoc,
  record.KeyedNamed<ShuntResistorLoc>
>({
  fieldKey: "shuntResistorLoc",
  fieldProps: { label: "Shunt resistor location" },
  inputProps: {
    resourceName: "shunt resistor location",
    data: [
      { key: "Default", name: "Default" },
      { key: "Internal", name: "Internal" },
      { key: "External", name: "External" },
    ],
  },
});

const ResistanceConfigField = Form.buildSelectField<string, record.KeyedNamed<string>>({
  fieldKey: "resistanceConfig",
  fieldProps: { label: "Resistance configuration" },
  inputProps: {
    resourceName: "resistance configuration",
    data: [
      { key: "2Wire", name: "2-Wire" },
      { key: "3Wire", name: "3-Wire" },
      { key: "4Wire", name: "4-Wire" },
    ],
  },
});

const StrainConfig = Form.buildSelectField({
  fieldKey: "strainConfig",
  fieldProps: { label: "Strain configuration" },
  inputProps: {
    resourceName: "strain configuration",
    data: [
      { key: "FullBridgeI", name: "Full bridge I" },
      { key: "FullBridgeII", name: "Full bridge II" },
      { key: "FullBridgeIII", name: "Full bridge III" },
      { key: "HalfBridgeI", name: "Half bridge I" },
      { key: "HalfBridgeII", name: "Half bridge II" },
      { key: "QuarterBridgeI", name: "Quarter bridge I" },
      { key: "QuarterBridgeII", name: "Quarter bridge II" },
    ],
  },
});

const SensitivityField = Form.buildNumericField({
  fieldKey: "sensitivity",
  fieldProps: { label: "Sensitivity" },
  inputProps: {},
});

const ForceUnitsField = Form.buildSelectField<
  ForceUnits,
  record.KeyedNamed<ForceUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Force units" },
  inputProps: {
    resourceName: "force units",
    data: [
      { key: "Newtons", name: "Newtons" },
      { key: "Pounds", name: "Pounds" },
      { key: "KilogramForce", name: "Kilograms" },
    ],
  },
});

const ElectricalUnitsField = Form.buildSelectField<
  ElectricalUnits,
  record.KeyedNamed<ElectricalUnits>
>({
  fieldKey: "electricalUnits",
  fieldProps: { label: "Electrical units" },
  inputProps: {
    resourceName: "electrical units",
    data: [
      { key: "VoltsPerVolt", name: "V/V" },
      { key: "mVoltsPerVolt", name: "mV/V" },
    ],
  },
});

const ChargeUnitsField = Form.buildSelectField<
  ChargeUnits,
  record.KeyedNamed<ChargeUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Charge units" },
  inputProps: {
    resourceName: "charge units",
    data: [
      { key: "Coulombs", name: "Coulombs" },
      { key: "PicoCoulombs", name: "Picocoulombs" },
    ],
  },
});

const PressureUnitsField = Form.buildSelectField<
  PressureUnits,
  record.KeyedNamed<PressureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Pressure units" },
  inputProps: {
    resourceName: "pressure units",
    data: [
      { key: "Pascals", name: "Pascals" },
      { key: "PoundsPerSquareInch", name: "PSI" },
    ],
  },
});

const TemperatureUnitsField = Form.buildSelectField<
  TemperatureUnits,
  record.KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
    data: [
      { key: "DegC", name: "Celsius" },
      { key: "DegF", name: "Fahrenheit" },
      { key: "Kelvins", name: "Kelvin" },
      { key: "DegR", name: "Rankine" },
    ],
  },
});

const ThermocoupleTypeField = Form.buildSelectField({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
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

const TorqueUnitsField = Form.buildSelectField<
  TorqueUnits,
  record.KeyedNamed<TorqueUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Torque units" },
  inputProps: {
    resourceName: "torque units",
    data: [
      { key: "NewtonMeters", name: "Newton meters" },
      { key: "InchOunces", name: "Inch ounces" },
      { key: "FootPounds", name: "Foot pounds" },
    ],
  },
});

const ForceSensitivityUnitsField = Form.buildSelectField({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: [
      { key: "mVoltsPerNewton", name: "mV/N" },
      { key: "mVoltsPerPound", name: "mV/lb" },
    ],
  },
});

const RTDTypeField = Form.buildSelectField({
  fieldKey: "rtdType",
  fieldProps: { label: "RTD type" },
  inputProps: {
    resourceName: "RTD type",
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

const ZERO_CJCS: Record<CJCType, CJC> = {
  built_in: { source: "built_in" },
  const_val: { source: "const_val", val: 0 },
  chan: { source: "chan", port: 0 },
};

const CJCSourceField = Form.buildSelectField<CJCType, PSelect.StaticEntry<CJCType>>({
  fieldKey: "cjc.source",
  fieldProps: {
    label: "CJC source",
    onChange: (value, { get, set, path }) => {
      if (get<CJCType>(path).value === value) return;
      set(path.slice(0, path.lastIndexOf(".")), ZERO_CJCS[value]);
    },
  },
  inputProps: {
    resourceName: "CJC source",
    data: [
      { key: "built_in", name: "Built in", icon: <Icon.Device /> },
      { key: "const_val", name: "Constant value", icon: <Icon.Constant /> },
      { key: "chan", name: "Channel", icon: <Icon.Channel /> },
    ],
  },
});

const VelocityUnitsField = Form.buildSelectField<
  VelocityUnits,
  record.KeyedNamed<VelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Velocity units" },
  inputProps: {
    resourceName: "velocity units",
    data: [
      { key: "MetersPerSecond", name: "m/s" },
      { key: "InchesPerSecond", name: "in/s" },
    ],
  },
});

const VelocitySensitivityUnitsField = Form.buildSelectField<
  VelocitySensitivityUnits,
  record.KeyedNamed<VelocitySensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: [
      { key: "MillivoltsPerMillimeterPerSecond", name: "mV/mm/s" },
      { key: "MilliVoltsPerInchPerSecond", name: "mV/in/s" },
    ],
  },
});

const CHANNEL_FORMS: Record<AIChannelType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <AccelUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <SensitivityField
        path={prefix}
        inputProps={{
          showDragHandle: false,
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
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_accel_4_wire_dc_voltage: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <AccelUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <SensitivityField
        path={prefix}
        inputProps={{
          showDragHandle: false,
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
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Form.SwitchField
        path={`${prefix}.useExcitForScaling`}
        label="Use excitation for scaling"
      />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_accel_charge: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <AccelUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <SensitivityField
        path={prefix}
        inputProps={{
          showDragHandle: false,
          children: (
            <AccelChargeSensitivityUnitsField
              path={prefix}
              grow
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_bridge: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <ElectricalUnitsField path={prefix} fieldKey="units" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_charge: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <ChargeUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_current: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ShuntResistorLocField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.extShuntResistorVal`}
          label="Shunt resistance"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_current_rms: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ShuntResistorLocField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.extShuntResistorVal`}
          label="Shunt resistance"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_bridge_polynomial: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <ForceUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      <CoefficientsField
        path={`${prefix}.forwardCoeffs`}
        label="Forward coefficients"
      />
      <CoefficientsField
        path={`${prefix}.reverseCoeffs`}
        label="Reverse coefficients"
      />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_bridge_table: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <ForceUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x gap="small">
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x gap="small">
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      {/* electricalVals */}
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_bridge_two_point_lin: ({ prefix }) => (
    <>
      <ForceUnitsField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        <ElectricalUnitsField grow path={prefix} />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.firstPhysicalVal`}
          label="Physical value one"
          grow
        />
        <Form.NumericField
          path={`${prefix}.secondPhysicalVal`}
          label="Physical value two"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.firstElectricalVal`}
          label="Electrical value one"
          grow
        />
        <Form.NumericField
          path={`${prefix}.secondElectricalVal`}
          label="Electrical value two"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_iepe: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <ForceUnitsField
        path={prefix}
        inputProps={{
          filter: ({ key }) => key !== "KilogramForce",
          resourceName: "force units",
        }}
      />
      <SensitivityField
        path={prefix}
        inputProps={{
          children: (
            <ForceSensitivityUnitsField
              path={prefix}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),

  ai_freq_voltage: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.thresholdLevel`}
          label="Threshold level"
          grow
        />
        <Form.NumericField path={`${prefix}.hysteresis`} label="Hysteresis" grow />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_microphone: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.micSensitivity`}
          label="Microphone sensitivity"
          grow
        />
        <Form.NumericField
          path={`${prefix}.maxSndPressLevel`}
          label="Max sound pressure level"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_pressure_bridge_polynomial: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <PressureUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      <CoefficientsField
        path={`${prefix}.forwardCoeffs`}
        label="Forward coefficients"
      />
      <CoefficientsField
        path={`${prefix}.reverseCoeffs`}
        label="Reverse coefficients"
      />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_pressure_bridge_table: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <PressureUnitsField path={prefix} />
      <BridgeConfigField path={prefix} />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Form.NumericField
        path={`${prefix}.nominalBridgeResistance`}
        label="Nominal bridge resistance"
      />
      <Flex.Box x>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      {/* electricalVals */}
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_pressure_bridge_two_point_lin: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <PressureUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
          grow
          className={CSS.BM("ni-field", "narrow")}
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
          className={CSS.BM("ni-field", "half")}
        />
        <ElectricalUnitsField
          path={prefix}
          grow
          className={CSS.BM("ni-field", "half")}
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.firstPhysicalVal`}
          label="Physical value one"
          grow
          className={CSS.BM("ni-field", "half")}
        />
        <Form.NumericField
          path={`${prefix}.secondPhysicalVal`}
          label="Physical value two"
          className={CSS.BM("ni-field", "half")}
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.firstElectricalVal`}
          label="Electrical value one"
          className={CSS.BM("ni-field", "half")}
          grow
        />
        <Form.NumericField
          path={`${prefix}.secondElectricalVal`}
          label="Electrical value two"
          className={CSS.BM("ni-field", "half")}
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_resistance: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <ResistanceConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_rtd: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TemperatureUnitsField path={prefix} grow />
        <RTDTypeField path={prefix} grow />
      </Flex.Box>
      <ResistanceConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
          grow
        />
      </Flex.Box>
      <Form.NumericField path={`${prefix}.r0`} label="R0 Resistance" grow />
    </>
  ),
  ai_strain_gauge: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <StrainConfig path={prefix} />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField path={`${prefix}.gageFactor`} label="Gage factor" grow />
        <Form.NumericField
          path={`${prefix}.initialBridgeVoltage`}
          label="Initial bridge voltage"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={`${prefix}.nominalGageResistance`}
          label="Nominal gage resistance"
          grow
        />

        <Form.NumericField
          path={`${prefix}.poissonRatio`}
          label="Poisson's Ratio"
          grow
        />
        <Form.NumericField
          path={`${prefix}.leadWireResistance`}
          label="Lead wire resistance"
          grow
        />
      </Flex.Box>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_temp_builtin: ({ prefix }) => <TemperatureUnitsField path={prefix} />,
  ai_thermistor_iex: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TemperatureUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <ResistanceConfigField path={prefix} />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <Form.NumericField path={`${prefix}.a`} label="Steinhart-Hart A" grow />
        <Form.NumericField path={`${prefix}.b`} label="Steinhart-Hart B" grow />
        <Form.NumericField path={`${prefix}.c`} label="Steinhart-Hart C" grow />
      </Flex.Box>
    </>
  ),
  ai_thermistor_vex: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TemperatureUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <ResistanceConfigField path={prefix} />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <Form.NumericField path={`${prefix}.a`} label="Steinhart-Hart A" grow />
        <Form.NumericField path={`${prefix}.b`} label="Steinhart-Hart B" grow />
        <Form.NumericField path={`${prefix}.c`} label="Steinhart-Hart C" grow />
      </Flex.Box>
      <Form.NumericField path={`${prefix}.r1`} label="Reference resistor" />
    </>
  ),
  ai_thermocouple: ({ prefix }) => {
    const cjcSource = Form.useFieldValue<CJCType>(`${prefix}.cjc.source`, {
      optional: true,
    });
    return (
      <>
        <MinMaxValueFields path={prefix} />
        <Flex.Box x>
          <TemperatureUnitsField path={prefix} grow />
          <ThermocoupleTypeField path={prefix} grow />
        </Flex.Box>
        <Flex.Box x>
          <CJCSourceField path={prefix} grow />
          {cjcSource === "const_val" && (
            <Form.NumericField path={`${prefix}.cjc.val`} label="CJC value" grow />
          )}
          {cjcSource === "chan" && (
            <Form.NumericField path={`${prefix}.cjc.port`} label="CJC port" grow />
          )}
        </Flex.Box>
      </>
    );
  },
  ai_torque_bridge_polynomial: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      <CoefficientsField
        path={`${prefix}.forwardCoeffs`}
        label="Forward coefficients"
      />
      <CoefficientsField
        path={`${prefix}.reverseCoeffs`}
        label="Reverse coefficients"
      />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_torque_bridge_table: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal bridge resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        {/* physicalVals */}
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      {/* electricalVals */}
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_torque_bridge_two_point_lin: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <BridgeConfigField path={prefix} />
      <Form.NumericField
        path={`${prefix}.nominalBridgeResistance`}
        label="Nominal bridge resistance"
      />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          grow
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x gap="small">
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          grow
          path={`${prefix}.firstPhysicalVal`}
          label="Physical value one"
        />
        <Form.NumericField
          grow
          path={`${prefix}.secondPhysicalVal`}
          label="Physical value two"
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          grow
          path={`${prefix}.firstElectricalVal`}
          label="Electrical value one"
        />
        <Form.NumericField
          grow
          path={`${prefix}.secondElectricalVal`}
          label="Electrical value two"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_velocity_iepe: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <VelocityUnitsField path={prefix} />
      <Form.NumericField
        path={`${prefix}.sensitivity`}
        label="Sensitivity"
        inputProps={{
          children: (
            <VelocitySensitivityUnitsField
              path={prefix}
              showLabel={false}
              showHelpText={false}
              inputProps={{
                triggerProps: {
                  style: { borderTopLeftRadius: 0, borderBottomLeftRadius: 0 },
                },
              }}
            />
          ),
        }}
      />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="currentExcitSource"
          label="Current excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current excitation value"
        />
      </Flex.Box>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_voltage: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_voltage_rms: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_voltage_with_excit: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
      <Divider.Divider x padded="bottom" />
      <BridgeConfigField path={prefix} />
      <Flex.Box x>
        <ExcitSourceField
          path={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage excitation source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage excitation value"
        />
      </Flex.Box>
      <Form.SwitchField
        path={`${prefix}.useExcitForScaling`}
        label="Use excitation for scaling"
      />
      <Divider.Divider x padded="bottom" />
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
      <Flex.Box x wrap>
        <Select path={`${prefix}.device`} />
        {type !== "ai_temp_builtin" && <PortField path={prefix} />}
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Form prefix={prefix} />
    </>
  );
};
