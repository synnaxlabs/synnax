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
import { selectData } from "@/feature/ni/task/selectData";
import {
  type AccelChargeSensitivityUnits,
  type AccelSensitivityUnits,
  type AccelUnits,
  type AIChannelType,
  type BridgeConfig,
  type ChargeUnits,
  type CJC,
  type CJCType,
  type ElectricalUnits,
  type ExcitationSource,
  type ForceSensitivityUnits,
  type ForceUnits,
  type PressureUnits,
  type ResistanceConfig,
  type RTDType,
  type ShuntResistorLoc,
  type StrainConfig,
  type TemperatureUnits,
  type TerminalConfig,
  type ThermocoupleType,
  type TorqueUnits,
  type VelocitySensitivityUnits,
  type VelocityUnits,
} from "@/feature/ni/task/types";
import { CSS } from "@/platform/css";

interface FormProps {
  prefix: string;
}

const TERMINAL_CONFIG_NAMES = {
  RSE: "Referenced single ended",
  NRSE: "Non-referenced single ended",
  Diff: "Differential",
  PseudoDiff: "Pseudo-Differential",
  Cfg_Default: "Default",
} as const satisfies Record<TerminalConfig, string>;

const TerminalConfigField = Form.buildSelectField<
  TerminalConfig,
  record.KeyedNamed<TerminalConfig>
>({
  fieldKey: "terminalConfig",
  fieldProps: { label: "Terminal configuration" },
  inputProps: {
    resourceName: "terminal configuration",
    data: selectData(TERMINAL_CONFIG_NAMES),
  },
});

const ACCEL_UNITS_NAMES = {
  g: "g",
  MetersPerSecondSquared: "m/s²",
  InchesPerSecondSquared: "in/s²",
} as const satisfies Record<AccelUnits, string>;

const AccelUnitsField = Form.buildSelectField<
  AccelUnits,
  record.KeyedNamed<AccelUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Acceleration units" },
  inputProps: {
    resourceName: "acceleration units",
    data: selectData(ACCEL_UNITS_NAMES),
  },
});

const ACCEL_SENSITIVITY_UNITS_NAMES = {
  mVoltsPerG: "mV/g",
  VoltsPerG: "V/g",
} as const satisfies Record<AccelSensitivityUnits, string>;

const AccelSensitivityUnitsField = Form.buildSelectField<
  AccelSensitivityUnits,
  record.KeyedNamed<AccelSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: selectData(ACCEL_SENSITIVITY_UNITS_NAMES),
  },
});

const ACCEL_CHARGE_SENSITIVITY_UNITS_NAMES = {
  PicoCoulombsPerG: "pC/g",
  PicoCoulombsPerMetersPerSecondSquared: "pC/(m/s²)",
  PicoCoulombsPerInchesPerSecondSquared: "pC/(in/s²)",
} as const satisfies Record<AccelChargeSensitivityUnits, string>;

const AccelChargeSensitivityUnitsField = Form.buildSelectField<
  AccelChargeSensitivityUnits,
  record.KeyedNamed<AccelChargeSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: selectData(ACCEL_CHARGE_SENSITIVITY_UNITS_NAMES),
  },
});

const EXCIT_SOURCE_NAMES = {
  Internal: "Internal",
  External: "External",
  None: "None",
} as const satisfies Record<ExcitationSource, string>;

const ExcitSourceField = Form.buildSelectField<
  ExcitationSource,
  record.KeyedNamed<ExcitationSource>
>({
  fieldKey: "excitSource",
  fieldProps: { label: "Excitation source" },
  inputProps: {
    resourceName: "excitation source",
    data: selectData(EXCIT_SOURCE_NAMES),
  },
});

const BRIDGE_CONFIG_NAMES = {
  FullBridge: "Full bridge",
  HalfBridge: "Half bridge",
  QuarterBridge: "Quarter bridge",
} as const satisfies Record<BridgeConfig, string>;

const BridgeConfigField = Form.buildSelectField<
  BridgeConfig,
  record.KeyedNamed<BridgeConfig>
>({
  fieldKey: "bridgeConfig",
  fieldProps: { label: "Bridge configuration" },
  inputProps: {
    resourceName: "bridge configuration",
    data: selectData(BRIDGE_CONFIG_NAMES),
  },
});

const SHUNT_RESISTOR_LOC_NAMES = {
  Default: "Default",
  Internal: "Internal",
  External: "External",
} as const satisfies Record<ShuntResistorLoc, string>;

const ShuntResistorLocField = Form.buildSelectField<
  ShuntResistorLoc,
  record.KeyedNamed<ShuntResistorLoc>
>({
  fieldKey: "shuntResistorLoc",
  fieldProps: { label: "Shunt resistor location" },
  inputProps: {
    resourceName: "shunt resistor location",
    data: selectData(SHUNT_RESISTOR_LOC_NAMES),
  },
});

const RESISTANCE_CONFIG_NAMES = {
  "2Wire": "2-Wire",
  "3Wire": "3-Wire",
  "4Wire": "4-Wire",
} as const satisfies Record<ResistanceConfig, string>;

const ResistanceConfigField = Form.buildSelectField<
  ResistanceConfig,
  record.KeyedNamed<ResistanceConfig>
>({
  fieldKey: "resistanceConfig",
  fieldProps: { label: "Resistance configuration" },
  inputProps: {
    resourceName: "resistance configuration",
    data: selectData(RESISTANCE_CONFIG_NAMES),
  },
});

const STRAIN_CONFIG_NAMES = {
  FullBridgeI: "Full bridge I",
  FullBridgeII: "Full bridge II",
  FullBridgeIII: "Full bridge III",
  HalfBridgeI: "Half bridge I",
  HalfBridgeII: "Half bridge II",
  QuarterBridgeI: "Quarter bridge I",
  QuarterBridgeII: "Quarter bridge II",
} as const satisfies Record<StrainConfig, string>;

const StrainConfigField = Form.buildSelectField<
  StrainConfig,
  record.KeyedNamed<StrainConfig>
>({
  fieldKey: "strainConfig",
  fieldProps: { label: "Strain configuration" },
  inputProps: {
    resourceName: "strain configuration",
    data: selectData(STRAIN_CONFIG_NAMES),
  },
});

const SensitivityField = Form.buildNumericField({
  fieldKey: "sensitivity",
  fieldProps: { label: "Sensitivity" },
  inputProps: {},
});

const FORCE_UNITS_NAMES = {
  Newtons: "Newtons",
  Pounds: "Pounds",
  KilogramForce: "Kilograms",
} as const satisfies Record<ForceUnits, string>;

const ForceUnitsField = Form.buildSelectField<
  ForceUnits,
  record.KeyedNamed<ForceUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Force units" },
  inputProps: {
    resourceName: "force units",
    data: selectData(FORCE_UNITS_NAMES),
  },
});

const ELECTRICAL_UNITS_NAMES = {
  VoltsPerVolt: "V/V",
  mVoltsPerVolt: "mV/V",
} as const satisfies Record<ElectricalUnits, string>;

const ElectricalUnitsField = Form.buildSelectField<
  ElectricalUnits,
  record.KeyedNamed<ElectricalUnits>
>({
  fieldKey: "electricalUnits",
  fieldProps: { label: "Electrical units" },
  inputProps: {
    resourceName: "electrical units",
    data: selectData(ELECTRICAL_UNITS_NAMES),
  },
});

const CHARGE_UNITS_NAMES = {
  Coulombs: "Coulombs",
  PicoCoulombs: "Picocoulombs",
} as const satisfies Record<ChargeUnits, string>;

const ChargeUnitsField = Form.buildSelectField<
  ChargeUnits,
  record.KeyedNamed<ChargeUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Charge units" },
  inputProps: {
    resourceName: "charge units",
    data: selectData(CHARGE_UNITS_NAMES),
  },
});

const PRESSURE_UNITS_NAMES = {
  Pascals: "Pascals",
  PoundsPerSquareInch: "PSI",
  Bar: "Bar",
} as const satisfies Record<PressureUnits, string>;

const PressureUnitsField = Form.buildSelectField<
  PressureUnits,
  record.KeyedNamed<PressureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Pressure units" },
  inputProps: {
    resourceName: "pressure units",
    data: selectData(PRESSURE_UNITS_NAMES),
  },
});

const TEMPERATURE_UNITS_NAMES = {
  DegC: "Celsius",
  DegF: "Fahrenheit",
  Kelvins: "Kelvin",
  DegR: "Rankine",
} as const satisfies Record<TemperatureUnits, string>;

const TemperatureUnitsField = Form.buildSelectField<
  TemperatureUnits,
  record.KeyedNamed<TemperatureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
    data: selectData(TEMPERATURE_UNITS_NAMES),
  },
});

const THERMOCOUPLE_TYPE_NAMES = {
  B: "B",
  E: "E",
  J: "J",
  K: "K",
  N: "N",
  R: "R",
  S: "S",
  T: "T",
} as const satisfies Record<ThermocoupleType, string>;

const ThermocoupleTypeField = Form.buildSelectField<
  ThermocoupleType,
  record.KeyedNamed<ThermocoupleType>
>({
  fieldKey: "thermocoupleType",
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
    data: selectData(THERMOCOUPLE_TYPE_NAMES),
  },
});

const TORQUE_UNITS_NAMES = {
  NewtonMeters: "Newton meters",
  InchOunces: "Inch ounces",
  InchPounds: "Inch pounds",
  FootPounds: "Foot pounds",
} as const satisfies Record<TorqueUnits, string>;

const TorqueUnitsField = Form.buildSelectField<
  TorqueUnits,
  record.KeyedNamed<TorqueUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Torque units" },
  inputProps: {
    resourceName: "torque units",
    data: selectData(TORQUE_UNITS_NAMES),
  },
});

const FORCE_SENSITIVITY_UNITS_NAMES = {
  mVoltsPerNewton: "mV/N",
  mVoltsPerPound: "mV/lb",
} as const satisfies Record<ForceSensitivityUnits, string>;

const ForceSensitivityUnitsField = Form.buildSelectField<
  ForceSensitivityUnits,
  record.KeyedNamed<ForceSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: selectData(FORCE_SENSITIVITY_UNITS_NAMES),
  },
});

const R_T_D_TYPE_NAMES = {
  Pt3750: "Pt3750",
  Pt3851: "Pt3851",
  Pt3911: "Pt3911",
  Pt3916: "Pt3916",
  Pt3920: "Pt3920",
  Pt3928: "Pt3928",
} as const satisfies Record<RTDType, string>;

const RTDTypeField = Form.buildSelectField<RTDType, record.KeyedNamed<RTDType>>({
  fieldKey: "rtdType",
  fieldProps: { label: "RTD type" },
  inputProps: {
    resourceName: "RTD type",
    data: selectData(R_T_D_TYPE_NAMES),
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

const VELOCITY_UNITS_NAMES = {
  MetersPerSecond: "m/s",
  InchesPerSecond: "in/s",
} as const satisfies Record<VelocityUnits, string>;

const VelocityUnitsField = Form.buildSelectField<
  VelocityUnits,
  record.KeyedNamed<VelocityUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Velocity units" },
  inputProps: {
    resourceName: "velocity units",
    data: selectData(VELOCITY_UNITS_NAMES),
  },
});

const VELOCITY_SENSITIVITY_UNITS_NAMES = {
  MillivoltsPerMillimeterPerSecond: "mV/mm/s",
  MilliVoltsPerInchPerSecond: "mV/in/s",
} as const satisfies Record<VelocitySensitivityUnits, string>;

const VelocitySensitivityUnitsField = Form.buildSelectField<
  VelocitySensitivityUnits,
  record.KeyedNamed<VelocitySensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: selectData(VELOCITY_SENSITIVITY_UNITS_NAMES),
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
        path={`${prefix}.scaledByExcitation`}
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
      <StrainConfigField path={prefix} />
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
        path={`${prefix}.scaledByExcitation`}
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
