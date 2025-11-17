// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Divider, Flex, Form, Icon } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
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
  type VelocitySensitivityUnits,
  type VelocityUnits,
} from "@/hardware/ni/task/types";

interface FormProps {
  prefix: string;
}

const TerminalConfigField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "terminalConfig",
  fieldProps: { label: "Terminal Configuration" },
  inputProps: {
    resourceName: "terminal configuration",
    data: [
      { key: "RSE", name: "Referenced Single Ended" },
      { key: "NRSE", name: "Non-Referenced Single Ended" },
      { key: "Diff", name: "Differential" },
      { key: "PseudoDiff", name: "Pseudo-Differential" },
      { key: "Cfg_Default", name: "Default" },
    ],
  },
});

const AccelSensitivityUnitsField = Form.buildSelectField<
  AccelSensitivityUnits,
  record.KeyedNamed<AccelSensitivityUnits>
>({
  fieldKey: "sensitivityUnits",
  fieldProps: { label: "Sensitivity Units" },
  inputProps: {
    resourceName: "sensitivity units",
    data: [
      { key: "mVoltsPerG", name: "mV/g" },
      { key: "VoltsPerG", name: "V/g" },
    ],
  },
});

const ExcitSourceField = Form.buildSelectField<string, record.KeyedNamed>({
  fieldKey: "excitSource",
  fieldProps: { label: "Excitation Source" },
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
  fieldProps: { label: "Bridge Configuration" },
  inputProps: {
    resourceName: "bridge configuration",
    data: [
      { key: "FullBridge", name: "Full Bridge" },
      { key: "HalfBridge", name: "Half Bridge" },
      { key: "QuarterBridge", name: "Quarter Bridge" },
    ],
  },
});

const ShuntResistorLocField = Form.buildSelectField<
  ShuntResistorLoc,
  record.KeyedNamed<ShuntResistorLoc>
>({
  fieldKey: "shuntResistorLoc",
  fieldProps: { label: "Shunt Resistor Location" },
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
  fieldProps: { label: "Resistance Configuration" },
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
  fieldProps: { label: "Strain Configuration" },
  inputProps: {
    resourceName: "strain configuration",
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
  inputProps: {},
});

const ForceUnitsField = Form.buildSelectField<
  ForceUnits,
  record.KeyedNamed<ForceUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Force Units" },
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
  fieldProps: { label: "Electrical Units" },
  inputProps: {
    resourceName: "electrical units",
    data: [
      { key: "VoltsPerVolt", name: "V/V" },
      { key: "mVoltsPerVolt", name: "mV/V" },
    ],
  },
});

const PressureUnitsField = Form.buildSelectField<
  PressureUnits,
  record.KeyedNamed<PressureUnits>
>({
  fieldKey: "units",
  fieldProps: { label: "Pressure Units" },
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
  fieldProps: { label: "Temperature Units" },
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
  fieldProps: { label: "Thermocouple Type" },
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
  fieldProps: { label: "Torque Units" },
  inputProps: {
    resourceName: "torque units",
    data: [
      { key: "NewtonMeters", name: "Newton Meters" },
      { key: "InchOunces", name: "Inch Ounces" },
      { key: "FootPounds", name: "Foot Pounds" },
    ],
  },
});

const CHANNEL_FORMS: Record<AIChannelType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <MinMaxValueFields path={prefix} />
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
          label="Current Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
      </Flex.Box>
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
          label="Nominal Bridge Resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
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
      </Flex.Box>
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
          label="Shunt Resistance"
          grow
        />
      </Flex.Box>
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
          label="Nominal Bridge Resistance"
          grow
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x gap="small">
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
      </Flex.Box>
      <Flex.Box x>
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
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
          label="Voltage Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <ForceUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        <ElectricalUnitsField grow path={prefix} />
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_force_iepe: ({ prefix }) => {
    const SensitivityUnits = Form.buildSelectField({
      fieldKey: "sensitivityUnits",
      fieldProps: { label: "Sensitivity Units" },
      inputProps: {
        resourceName: "sensitivity units",
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
              <SensitivityUnits path={prefix} showLabel={false} showHelpText={false} />
            ),
          }}
        />
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
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
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },

  ai_microphone: ({ prefix }) => {
    const UnitsField = Form.buildSelectField({
      fieldKey: "units",
      fieldProps: { label: "Sound Pressure Units" },
      inputProps: {
        resourceName: "sound pressure units",
        data: [{ key: "Pascals", name: "Pascals" }],
      },
    });
    return (
      <>
        <TerminalConfigField path={prefix} />
        <UnitsField path={prefix} />
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
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
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <Flex.Box x>
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
        </Flex.Box>
        <Divider.Divider x padded="bottom" />
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_pressure_bridge_table: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <PressureUnitsField path={prefix} />
      <BridgeConfigField path={prefix} />
      <Flex.Box x>
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
      </Flex.Box>
      <Form.NumericField
        path={`${prefix}.nominalBridgeResistance`}
        label="Nominal Bridge Resistance"
      />
      <Flex.Box x>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
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
          label="Nominal Bridge Resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
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
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <PressureUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
          style={{ width: "50%" }}
        />
        <ElectricalUnitsField path={prefix} grow style={{ width: "50%" }} />
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <Flex.Box x>
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
          label="Current Excitation Source"
          grow
        />
        <Form.NumericField
          path={`${prefix}.currentExcitVal`}
          label="Current Excitation Value"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_rtd: ({ prefix }) => {
    const RTDTypeField = Form.buildSelectField({
      fieldKey: "rtdType",
      fieldProps: { label: "RTD Type" },
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
    return (
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
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
            grow
          />
        </Flex.Box>
        <Form.NumericField path={`${prefix}.r0`} label="R0 Resistance" grow />
      </>
    );
  },
  ai_strain_gauge: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <StrainConfig path={prefix} />
      <Flex.Box x>
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
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" grow />
        <Form.NumericField
          path={`${prefix}.initialBridgeVoltage`}
          label="Initial Bridge Voltage"
          grow
        />
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_temp_builtin: ({ prefix }) => <TemperatureUnitsField path={prefix} />,
  ai_thermocouple: ({ prefix }) => {
    const CJCSourceField = Form.buildSelectField({
      fieldKey: "cjcSource",
      fieldProps: {
        label: "CJC Source",
        onChange: (value, { set }) => {
          if (value === "ConstVal") set(`${prefix}.cjcVal`, 0);
          else if (value === "Chan") set(`${prefix}.cjcPort`, 0);
        },
      },
      inputProps: {
        resourceName: "CJC source",
        data: [
          { key: "BuiltIn", name: "Built In", icon: <Icon.Device /> },
          { key: "ConstVal", name: "Constant Value", icon: <Icon.Constant /> },
          { key: "Chan", name: "Channel", icon: <Icon.Channel /> },
        ],
      },
    });
    const cjcSource = Form.useFieldValue<string>(`${prefix}.cjcSource`, {
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
          {cjcSource === "ConstVal" && (
            <Form.NumericField path={`${prefix}.cjcVal`} label="CJC Value" grow />
          )}
          {cjcSource === "Chan" && (
            <Form.NumericField path={`${prefix}.cjcPort`} label="CJC Port" grow />
          )}
        </Flex.Box>
      </>
    );
  },
  ai_torque_bridge_table: ({ prefix }) => (
    <>
      <MinMaxValueFields path={prefix} />
      <TorqueUnitsField path={prefix} />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <BridgeConfigField path={prefix} grow />
        <Form.NumericField
          path={`${prefix}.nominalBridgeResistance`}
          label="Nominal Bridge Resistance"
        />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
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
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
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
        label="Nominal Bridge Resistance"
      />
      <Divider.Divider x padded="bottom" />
      <Flex.Box x>
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
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Flex.Box x gap="small">
        <TorqueUnitsField
          path={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
          grow
        />
        <ElectricalUnitsField path={prefix} grow />
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <Flex.Box x>
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
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <CustomScaleForm prefix={prefix} />
    </>
  ),
  ai_velocity_iepe: ({ prefix }) => {
    const VelocityUnits = Form.buildSelectField<
      VelocityUnits,
      record.KeyedNamed<VelocityUnits>
    >({
      fieldKey: "units",
      fieldProps: { label: "Velocity Units" },
      inputProps: {
        resourceName: "velocity units",
        data: [
          { key: "MetersPerSecond", name: "m/s" },
          { key: "InchesPerSecond", name: "in/s" },
        ],
      },
    });
    const SensitivityUnits = Form.buildSelectField<
      VelocitySensitivityUnits,
      record.KeyedNamed<VelocitySensitivityUnits>
    >({
      fieldKey: "sensitivityUnits",
      fieldProps: { label: "Sensitivity Units" },
      inputProps: {
        resourceName: "sensitivity units",
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
              <SensitivityUnits
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
            label="Current Excitation Source"
            grow
          />
          <Form.NumericField
            path={`${prefix}.currentExcitVal`}
            label="Current Excitation Value"
          />
        </Flex.Box>
        <CustomScaleForm prefix={prefix} />
      </>
    );
  },
  ai_voltage: ({ prefix }) => (
    <>
      <TerminalConfigField path={prefix} />
      <MinMaxValueFields path={prefix} />
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
        <Device.Select path={`${prefix}.device`} />
        <Device.PortField path={prefix} />
      </Flex.Box>
      <Divider.Divider x padded="bottom" />
      <Form prefix={prefix} />
    </>
  );
};
