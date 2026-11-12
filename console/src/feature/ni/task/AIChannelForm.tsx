// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ni } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as PSelect } from "@synnaxlabs/lyra/select";
import { deep, record } from "@synnaxlabs/x";
import { type FC, type ReactNode } from "react";

import { Select } from "@/feature/ni/device/Select";
import { createTypeFields } from "@/feature/ni/task/createTypeFields";
import { CustomScaleSection } from "@/feature/ni/task/CustomScaleForm";
import { MinMaxValueFields } from "@/feature/ni/task/MinMaxValueFields";
import { PortField } from "@/feature/ni/task/PortField";
import { accepts, ANALOG_INPUT } from "@/feature/ni/task/ports";
import {
  channelTypeItems,
  SelectAIChannelTypeField,
} from "@/feature/ni/task/SelectAIChannelTypeField";
import {
  type AccelChargeSensitivityUnits,
  type AccelSensitivityUnits,
  type AccelUnits,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
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
  TERMINAL_CONFIG_NAMES,
  TERMINAL_CONFIGS,
  type TerminalConfig,
  type ThermocoupleType,
  type TorqueUnits,
  type VelocitySensitivityUnits,
  type VelocityUnits,
} from "@/feature/ni/task/types";
import {
  allowedTypes,
  useAnalogInputPort,
  useChannelDevice,
} from "@/feature/ni/task/usePort";
import { Task } from "@/platform/task";

type Variant<K extends AIChannelType> = Extract<AIChannel, { type: K }>;
interface FormProps<K extends AIChannelType = AIChannelType> {
  path: deep.Path<Variant<K>>;
}

const SelectTerminalConfigField = Form.buildSelectField<TerminalConfig>({
  fieldProps: { label: "Terminal configuration" },
  inputProps: {
    resourceName: "terminal configuration",
    searchName: "configuration",
    children: Task.selectItems(TERMINAL_CONFIG_NAMES),
  },
});

// The configurations the port takes, and "Default" named by what it gives there.
const terminalConfigItems = (port: ni.AnalogInputPort, current: TerminalConfig) =>
  TERMINAL_CONFIGS.filter(
    (c) => c === "Cfg_Default" || c === current || accepts(port.terminalConfigs, c),
  ).map((c) => {
    const resolved = port.defaultTerminalConfig;
    return (
      <PSelect.Item<TerminalConfig> key={c} itemKey={c}>
        {c !== "Cfg_Default" || resolved === "Cfg_Default"
          ? TERMINAL_CONFIG_NAMES[c]
          : `${TERMINAL_CONFIG_NAMES[c]} (${TERMINAL_CONFIG_NAMES[resolved]})`}
      </PSelect.Item>
    );
  });

const TerminalConfigField = ({
  path,
}: {
  path: deep.Path<{ terminalConfig: TerminalConfig; device: string; port: number }>;
}) => {
  const port = useAnalogInputPort(path);
  const current = Form.useFieldValue(path.terminalConfig);
  if (port == null) return <SelectTerminalConfigField path={path.terminalConfig} />;
  return (
    <SelectTerminalConfigField
      path={path.terminalConfig}
      inputProps={{ children: terminalConfigItems(port, current) }}
    />
  );
};

const ACCEL_UNITS_NAMES = {
  g: "g",
  MetersPerSecondSquared: "m/s²",
  InchesPerSecondSquared: "in/s²",
} as const satisfies Record<AccelUnits, string>;

const AccelUnitsField = Form.buildSelectField<AccelUnits>({
  fieldProps: { label: "Acceleration units" },
  inputProps: {
    resourceName: "acceleration units",
    searchName: "unit",
    children: Task.selectItems(ACCEL_UNITS_NAMES),
  },
});

const ACCEL_SENSITIVITY_UNITS_NAMES = {
  mVoltsPerG: "mV/g",
  VoltsPerG: "V/g",
} as const satisfies Record<AccelSensitivityUnits, string>;

const AccelSensitivityUnitsField = Form.buildSelectField<AccelSensitivityUnits>({
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    searchName: "unit",
    children: Task.selectItems(ACCEL_SENSITIVITY_UNITS_NAMES),
  },
});

const ACCEL_CHARGE_SENSITIVITY_UNITS_NAMES = {
  PicoCoulombsPerG: "pC/g",
  PicoCoulombsPerMetersPerSecondSquared: "pC/(m/s²)",
  PicoCoulombsPerInchesPerSecondSquared: "pC/(in/s²)",
} as const satisfies Record<AccelChargeSensitivityUnits, string>;

const AccelChargeSensitivityUnitsField =
  Form.buildSelectField<AccelChargeSensitivityUnits>({
    fieldProps: { label: "Sensitivity units" },
    inputProps: {
      resourceName: "sensitivity units",
      searchName: "unit",
      children: Task.selectItems(ACCEL_CHARGE_SENSITIVITY_UNITS_NAMES),
    },
  });

const EXCIT_SOURCE_NAMES = {
  Internal: "Internal",
  External: "External",
  None: "None",
} as const satisfies Record<ExcitationSource, string>;

const ExcitSourceField = Form.buildSelectField<ExcitationSource>({
  fieldProps: { label: "Excitation source" },
  inputProps: {
    resourceName: "excitation source",
    searchName: "source",
    children: Task.selectItems(EXCIT_SOURCE_NAMES),
  },
});

const BRIDGE_CONFIG_NAMES = {
  FullBridge: "Full bridge",
  HalfBridge: "Half bridge",
  QuarterBridge: "Quarter bridge",
} as const satisfies Record<BridgeConfig, string>;

const BridgeConfigField = Form.buildSelectField<BridgeConfig>({
  fieldProps: { label: "Bridge configuration" },
  inputProps: {
    resourceName: "bridge configuration",
    searchName: "configuration",
    children: Task.selectItems(BRIDGE_CONFIG_NAMES),
  },
});

const SHUNT_RESISTOR_LOC_NAMES = {
  Default: "Default",
  Internal: "Internal",
  External: "External",
} as const satisfies Record<ShuntResistorLoc, string>;

const ShuntResistorLocField = Form.buildSelectField<ShuntResistorLoc>({
  fieldProps: { label: "Shunt resistor location" },
  inputProps: {
    resourceName: "shunt resistor location",
    searchName: "location",
    children: Task.selectItems(SHUNT_RESISTOR_LOC_NAMES),
  },
});

const RESISTANCE_CONFIG_NAMES = {
  "2Wire": "2-Wire",
  "3Wire": "3-Wire",
  "4Wire": "4-Wire",
} as const satisfies Record<ResistanceConfig, string>;

const ResistanceConfigField = Form.buildSelectField<ResistanceConfig>({
  fieldProps: { label: "Resistance configuration" },
  inputProps: {
    resourceName: "resistance configuration",
    searchName: "configuration",
    children: Task.selectItems(RESISTANCE_CONFIG_NAMES),
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

const StrainConfigField = Form.buildSelectField<StrainConfig>({
  fieldProps: { label: "Strain configuration" },
  inputProps: {
    resourceName: "strain configuration",
    searchName: "configuration",
    children: Task.selectItems(STRAIN_CONFIG_NAMES),
  },
});

const SensitivityField = Form.buildNumericField({
  fieldProps: { label: "Sensitivity" },
  inputProps: {},
});

const FORCE_UNITS_NAMES = {
  Newtons: "Newtons",
  Pounds: "Pounds",
  KilogramForce: "Kilograms",
} as const satisfies Record<ForceUnits, string>;

const IEPE_FORCE_UNITS_NAMES = record.omit(FORCE_UNITS_NAMES, "KilogramForce");

const ForceUnitsField = Form.buildSelectField<ForceUnits>({
  fieldProps: { label: "Force units" },
  inputProps: {
    resourceName: "force units",
    children: Task.selectItems(FORCE_UNITS_NAMES),
  },
});

const ELECTRICAL_UNITS_NAMES = {
  VoltsPerVolt: "V/V",
  mVoltsPerVolt: "mV/V",
} as const satisfies Record<ElectricalUnits, string>;

const ElectricalUnitsField = Form.buildSelectField<ElectricalUnits>({
  fieldProps: { label: "Electrical units" },
  inputProps: {
    resourceName: "electrical units",
    searchName: "unit",
    children: Task.selectItems(ELECTRICAL_UNITS_NAMES),
  },
});

const CHARGE_UNITS_NAMES = {
  Coulombs: "Coulombs",
  PicoCoulombs: "Picocoulombs",
} as const satisfies Record<ChargeUnits, string>;

const ChargeUnitsField = Form.buildSelectField<ChargeUnits>({
  fieldProps: { label: "Charge units" },
  inputProps: {
    resourceName: "charge units",
    children: Task.selectItems(CHARGE_UNITS_NAMES),
  },
});

const PRESSURE_UNITS_NAMES = {
  Pascals: "Pascals",
  PoundsPerSquareInch: "PSI",
  Bar: "Bar",
} as const satisfies Record<PressureUnits, string>;

const PressureUnitsField = Form.buildSelectField<PressureUnits>({
  fieldProps: { label: "Pressure units" },
  inputProps: {
    resourceName: "pressure units",
    searchName: "unit",
    children: Task.selectItems(PRESSURE_UNITS_NAMES),
  },
});

const TEMPERATURE_UNITS_NAMES = {
  DegC: "Celsius",
  DegF: "Fahrenheit",
  Kelvins: "Kelvin",
  DegR: "Rankine",
} as const satisfies Record<TemperatureUnits, string>;

const TEMPERATURE_UNIT_SYMBOLS = {
  DegC: "°C",
  DegF: "°F",
  Kelvins: "K",
  DegR: "°R",
} as const satisfies Record<TemperatureUnits, string>;

const TemperatureUnitsField = Form.buildSelectField<TemperatureUnits>({
  fieldProps: { label: "Temperature units" },
  inputProps: {
    resourceName: "temperature units",
    searchName: "unit",
    children: Task.selectItems(TEMPERATURE_UNITS_NAMES),
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

const ThermocoupleTypeField = Form.buildSelectField<ThermocoupleType>({
  fieldProps: { label: "Thermocouple type" },
  inputProps: {
    resourceName: "thermocouple type",
    searchName: "type",
    children: Task.selectItems(THERMOCOUPLE_TYPE_NAMES),
  },
});

const TORQUE_UNITS_NAMES = {
  NewtonMeters: "Newton meters",
  InchOunces: "Inch ounces",
  InchPounds: "Inch pounds",
  FootPounds: "Foot pounds",
} as const satisfies Record<TorqueUnits, string>;

const TorqueUnitsField = Form.buildSelectField<TorqueUnits>({
  fieldProps: { label: "Torque units" },
  inputProps: {
    resourceName: "torque units",
    children: Task.selectItems(TORQUE_UNITS_NAMES),
  },
});

const FORCE_SENSITIVITY_UNITS_NAMES = {
  mVoltsPerNewton: "mV/N",
  mVoltsPerPound: "mV/lb",
} as const satisfies Record<ForceSensitivityUnits, string>;

const ForceSensitivityUnitsField = Form.buildSelectField<ForceSensitivityUnits>({
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    searchName: "unit",
    children: Task.selectItems(FORCE_SENSITIVITY_UNITS_NAMES),
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

const RTDTypeField = Form.buildSelectField<RTDType>({
  fieldProps: { label: "RTD type" },
  inputProps: {
    resourceName: "RTD type",
    children: Task.selectItems(R_T_D_TYPE_NAMES),
  },
});

const ZERO_CJCS: Record<CJCType, CJC> = {
  built_in: { source: "built_in" },
  const_val: { source: "const_val", val: 0 },
  chan: { source: "chan", port: 0 },
};

const SelectCJCSource = Form.buildSelectField<CJCType>({
  fieldProps: { label: "CJC source" },
  inputProps: {
    resourceName: "CJC source",
    children: (
      <>
        <PSelect.Item itemKey="built_in">
          <Icon.Device />
          Built in
        </PSelect.Item>
        <PSelect.Item itemKey="const_val">
          <Icon.Constant />
          Constant value
        </PSelect.Item>
        <PSelect.Item itemKey="chan">
          <Icon.Channel />
          Channel
        </PSelect.Item>
      </>
    ),
  },
});

const CJCSourceField = ({ path }: { path: deep.Path<CJC> }) => (
  <SelectCJCSource
    path={path.source}
    onChange={(value, { get, set }) => {
      if (get(path.source).value === value) return;
      set(path, ZERO_CJCS[value]);
    }}
  />
);

const VELOCITY_UNITS_NAMES = {
  MetersPerSecond: "m/s",
  InchesPerSecond: "in/s",
} as const satisfies Record<VelocityUnits, string>;

const VelocityUnitsField = Form.buildSelectField<VelocityUnits>({
  fieldProps: { label: "Velocity units" },
  inputProps: {
    resourceName: "velocity units",
    searchName: "unit",
    children: Task.selectItems(VELOCITY_UNITS_NAMES),
  },
});

const VELOCITY_SENSITIVITY_UNITS_NAMES = {
  MillivoltsPerMillimeterPerSecond: "mV/mm/s",
  MilliVoltsPerInchPerSecond: "mV/in/s",
} as const satisfies Record<VelocitySensitivityUnits, string>;

const VelocitySensitivityUnitsField = Form.buildSelectField<VelocitySensitivityUnits>({
  fieldProps: { label: "Sensitivity units" },
  inputProps: {
    resourceName: "sensitivity units",
    searchName: "unit",
    children: Task.selectItems(VELOCITY_SENSITIVITY_UNITS_NAMES),
  },
});

const ThermocoupleForm: FC<FormProps<"ai_thermocouple">> = ({ path }) => {
  const cjcSource = Form.useFieldValue(path.cjc.source, {
    optional: true,
  });
  const units = Form.useFieldValue(path.units, {
    optional: true,
  });
  return (
    <>
      <ThermocoupleTypeField path={path.thermocoupleType} />
      <CJCSourceField path={path.cjc} />
      {cjcSource === "const_val" && (
        <Form.NumericField
          path={deep.variant(path.cjc, "source", "const_val").val}
          label="CJC value"
        />
      )}
      {cjcSource === "chan" && (
        <Form.NumericField
          path={deep.variant(path.cjc, "source", "chan").port}
          label="CJC port"
        />
      )}
      <TemperatureUnitsField path={path.units} />
      <MinMaxValueFields
        path={path}
        units={units == null ? undefined : TEMPERATURE_UNIT_SYMBOLS[units]}
      />
    </>
  );
};

const CHANNEL_FORMS: { [K in AIChannelType]: FC<FormProps<K>> } = {
  ai_accel: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <AccelUnitsField path={path.units} />
      <SensitivityField
        path={path.sensitivity}
        inputProps={{
          showDragHandle: false,
          children: (
            <AccelSensitivityUnitsField
              path={path.sensitivityUnits}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
    </>
  ),
  ai_accel_4_wire_dc_voltage: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <AccelUnitsField path={path.units} />
      <SensitivityField
        path={path.sensitivity}
        inputProps={{
          showDragHandle: false,
          children: (
            <AccelSensitivityUnitsField
              path={path.sensitivityUnits}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <Form.CheckboxField
        path={path.scaledByExcitation}
        label="Use excitation for scaling"
      />
    </>
  ),
  ai_accel_charge: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <AccelUnitsField path={path.units} />
      <SensitivityField
        path={path.sensitivity}
        inputProps={{
          showDragHandle: false,
          children: (
            <AccelChargeSensitivityUnitsField
              path={path.sensitivityUnits}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
    </>
  ),
  ai_bridge: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <ElectricalUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
    </>
  ),
  ai_charge: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <ChargeUnitsField path={path.units} />
    </>
  ),
  ai_current: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <ShuntResistorLocField path={path.shuntResistorLoc} />
      <Form.NumericField path={path.extShuntResistorVal} label="Shunt resistance" />
    </>
  ),
  ai_current_rms: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <ShuntResistorLocField path={path.shuntResistorLoc} />
      <Form.NumericField path={path.extShuntResistorVal} label="Shunt resistance" />
    </>
  ),
  ai_force_bridge_polynomial: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <ForceUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <ForceUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Task.CoefficientsField path={path.forwardCoeffs} label="Forward coefficients" />
      <Task.CoefficientsField path={path.reverseCoeffs} label="Reverse coefficients" />
    </>
  ),
  ai_force_bridge_table: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <ForceUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <ForceUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
    </>
  ),
  ai_force_bridge_two_point_lin: ({ path }) => (
    <>
      <ForceUnitsField path={path.units} />
      <MinMaxValueFields path={path} />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ForceUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Form.NumericField path={path.firstPhysicalVal} label="Physical value one" />
      <Form.NumericField path={path.secondPhysicalVal} label="Physical value two" />
      <Form.NumericField path={path.firstElectricalVal} label="Electrical value one" />
      <Form.NumericField path={path.secondElectricalVal} label="Electrical value two" />
    </>
  ),
  ai_force_iepe: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <ForceUnitsField
        path={path.units}
        inputProps={{ children: Task.selectItems(IEPE_FORCE_UNITS_NAMES) }}
      />
      <SensitivityField
        path={path.sensitivity}
        inputProps={{
          children: (
            <ForceSensitivityUnitsField
              path={path.sensitivityUnits}
              showLabel={false}
              showHelpText={false}
            />
          ),
        }}
      />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
    </>
  ),

  ai_freq_voltage: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <Form.NumericField path={path.thresholdLevel} label="Threshold level" />
      <Form.NumericField path={path.hysteresis} label="Hysteresis" />
    </>
  ),
  ai_microphone: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <Form.NumericField path={path.micSensitivity} label="Microphone sensitivity" />
      <Form.NumericField
        path={path.maxSndPressLevel}
        label="Max sound pressure level"
      />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
    </>
  ),
  ai_pressure_bridge_polynomial: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <PressureUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <PressureUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Task.CoefficientsField path={path.forwardCoeffs} label="Forward coefficients" />
      <Task.CoefficientsField path={path.reverseCoeffs} label="Reverse coefficients" />
    </>
  ),
  ai_pressure_bridge_table: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <PressureUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <PressureUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
    </>
  ),
  ai_pressure_bridge_two_point_lin: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <PressureUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <PressureUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Form.NumericField path={path.firstPhysicalVal} label="Physical value one" />
      <Form.NumericField path={path.secondPhysicalVal} label="Physical value two" />
      <Form.NumericField path={path.firstElectricalVal} label="Electrical value one" />
      <Form.NumericField path={path.secondElectricalVal} label="Electrical value two" />
    </>
  ),
  ai_resistance: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <ResistanceConfigField path={path.resistanceConfig} />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
    </>
  ),
  ai_rtd: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TemperatureUnitsField path={path.units} />
      <RTDTypeField path={path.rtdType} />
      <ResistanceConfigField path={path.resistanceConfig} />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
      <Form.NumericField path={path.r0} label="R0 resistance" />
    </>
  ),
  ai_strain_gauge: ({ path }) => (
    <>
      <StrainConfigField path={path.strainConfig} />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField
        path={path.voltageExcitVal}
        label="Voltage excitation value"
        inputProps={{ endContent: "V" }}
      />
      <Form.NumericField path={path.gageFactor} label="Gage factor" />
      <Form.NumericField
        path={path.nominalGageResistance}
        label="Nominal gage resistance"
        inputProps={{ endContent: "Ω" }}
      />
      <Form.NumericField path={path.poissonRatio} label="Poisson's ratio" />
      <Form.NumericField
        path={path.leadWireResistance}
        label="Lead wire resistance"
        inputProps={{ endContent: "Ω" }}
      />
      <Form.NumericField
        path={path.initialBridgeVoltage}
        label="Initial bridge voltage"
        inputProps={{ endContent: "V" }}
      />
      <MinMaxValueFields path={path} units="strain" />
    </>
  ),
  ai_temp_builtin: ({ path }) => <TemperatureUnitsField path={path.units} />,
  ai_thermistor_iex: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TemperatureUnitsField path={path.units} />
      <ResistanceConfigField path={path.resistanceConfig} />
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
      <Form.NumericField path={path.a} label="Steinhart-Hart A" />
      <Form.NumericField path={path.b} label="Steinhart-Hart B" />
      <Form.NumericField path={path.c} label="Steinhart-Hart C" />
    </>
  ),
  ai_thermistor_vex: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TemperatureUnitsField path={path.units} />
      <ResistanceConfigField path={path.resistanceConfig} />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <Form.NumericField path={path.a} label="Steinhart-Hart A" />
      <Form.NumericField path={path.b} label="Steinhart-Hart B" />
      <Form.NumericField path={path.c} label="Steinhart-Hart C" />
      <Form.NumericField path={path.r1} label="Reference resistor" />
    </>
  ),
  ai_thermocouple: ThermocoupleForm,
  ai_torque_bridge_polynomial: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TorqueUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <TorqueUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Task.CoefficientsField path={path.forwardCoeffs} label="Forward coefficients" />
      <Task.CoefficientsField path={path.reverseCoeffs} label="Reverse coefficients" />
    </>
  ),
  ai_torque_bridge_table: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TorqueUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <TorqueUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
    </>
  ),
  ai_torque_bridge_two_point_lin: ({ path }) => (
    <>
      <MinMaxValueFields path={path} />
      <TorqueUnitsField path={path.units} />
      <BridgeConfigField path={path.bridgeConfig} />
      <Form.NumericField
        path={path.nominalBridgeResistance}
        label="Nominal bridge resistance"
      />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <TorqueUnitsField path={path.physicalUnits} label="Physical units" />
      <ElectricalUnitsField path={path.electricalUnits} />
      <Form.NumericField path={path.firstPhysicalVal} label="Physical value one" />
      <Form.NumericField path={path.secondPhysicalVal} label="Physical value two" />
      <Form.NumericField path={path.firstElectricalVal} label="Electrical value one" />
      <Form.NumericField path={path.secondElectricalVal} label="Electrical value two" />
    </>
  ),
  ai_velocity_iepe: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <VelocityUnitsField path={path.units} />
      <Form.NumericField
        path={path.sensitivity}
        label="Sensitivity"
        inputProps={{
          children: (
            <VelocitySensitivityUnitsField
              path={path.sensitivityUnits}
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
      <ExcitSourceField
        path={path.currentExcitSource}
        label="Current excitation source"
      />
      <Form.NumericField path={path.currentExcitVal} label="Current excitation value" />
    </>
  ),
  ai_voltage: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
    </>
  ),
  ai_voltage_rms: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
    </>
  ),
  ai_voltage_with_excit: ({ path }) => (
    <>
      <TerminalConfigField path={path} />
      <MinMaxValueFields path={path} />
      <BridgeConfigField path={path.bridgeConfig} />
      <ExcitSourceField
        path={path.voltageExcitSource}
        label="Voltage excitation source"
      />
      <Form.NumericField path={path.voltageExcitVal} label="Voltage excitation value" />
      <Form.CheckboxField
        path={path.scaledByExcitation}
        label="Use excitation for scaling"
      />
    </>
  ),
};

const AI_CHANNEL_TYPES = record.keys(AI_CHANNEL_TYPE_NAMES);
const PORTLESS_TYPES: AIChannelType[] = ["ai_temp_builtin"];
const deviceOf = (c: AIChannel) => c.device;
export interface AIChannelFormProps {
  type: AIChannelType;
  path: deep.Path<AIChannel>;
  /** The path of the task's channels. */
  channelsPath: deep.Target<AIChannel[]>;
}

const TypeFields = createTypeFields<AIChannel>(CHANNEL_FORMS);

export interface AIChannelSettingsProps {
  type: AIChannelType;
  path: deep.Path<AIChannel>;
  /** Shown first in the signal section. */
  children?: ReactNode;
}

/** The signal fields of the type, and the custom scale. */
export const AIChannelSettings = ({ type, path, children }: AIChannelSettingsProps) => (
  <>
    <Form.Section title="Signal">
      {children}
      <TypeFields type={type} path={path} />
    </Form.Section>
    <CustomScaleSection path={path} />
  </>
);

export const AIChannelForm = ({ type, path, channelsPath }: AIChannelFormProps) => {
  const device = useChannelDevice(path);
  const port = useAnalogInputPort(path);
  const types = allowedTypes(AI_CHANNEL_TYPES, type, port, PORTLESS_TYPES);
  return (
    <Form.Sections>
      <Form.Section title="Source">
        <Select path={path.device} />
        {type !== "ai_temp_builtin" && (
          <PortField
            path={deep.variant(path, "type", type)}
            channelsPath={channelsPath}
            device={device}
            kind={ANALOG_INPUT}
            deviceOf={deviceOf}
          />
        )}
      </Form.Section>
      <AIChannelSettings type={type} path={path}>
        <SelectAIChannelTypeField
          path={path}
          inputProps={{ allowNone: false, children: channelTypeItems(types) }}
        />
      </AIChannelSettings>
    </Form.Sections>
  );
};
