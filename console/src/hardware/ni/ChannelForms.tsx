import { Form, Channel, List, Select, Input, Align } from "@synnaxlabs/pluto";
import {
  AIChan,
  AIChanType,
  AccelSensitivityUnits,
  AccelerationUnits,
  ElectricalUnits,
  ExcitationSource,
  ForceUnits,
  TerminalConfig,
  TorqueUnits,
} from "@/hardware/ni/types";
import { FC, ReactElement } from "react";

interface FormProps {
  prefix: string;
  fieldKey?: string;
  label?: string;
}

const ChannelField = ({
  prefix,
  fieldKey = "channel",
  label = "Synnax Channel",
}: FormProps): ReactElement => (
  <Form.Field path={`${prefix}.${fieldKey}`} label={label}>
    {(p) => <Channel.SelectSingle {...p} />}
  </Form.Field>
);

interface NamedKey<K extends string = string> {
  key: K;
  name: string;
}

const NAMED_KEY_COLS: List.ColumnSpec<string, NamedKey>[] = [
  {
    key: "name",
    name: "Name",
  },
];

interface BuiltFieldProps<K extends string> extends FormProps {
  omit?: NamedKey<K>[];
}

const buildNamedKeySelect =
  <K extends string>(
    label_: string,
    fieldKey_: string,
    data: NamedKey<K>[],
  ): FC<BuiltFieldProps<K>> =>
  ({ prefix, fieldKey = fieldKey_, label = label_, omit }): ReactElement => {
    if (omit != null) data = data.filter((d) => !omit.includes(d));
    return (
      <Form.Field<K> path={`${prefix}.${fieldKey}`} label={label}>
        {(p) => (
          <Select.Single<K, NamedKey<K>>
            {...p}
            columns={NAMED_KEY_COLS}
            data={data}
            entryRenderKey="name"
          />
        )}
      </Form.Field>
    );
  };

const buildNamedKeySelectMultiple =
  <K extends string>(
    label_: string,
    fieldKey_: string,
    data: NamedKey<K>[],
  ): FC<BuiltFieldProps<K>> =>
  ({ prefix, fieldKey = fieldKey_, label = label_, omit }): ReactElement => {
    if (omit != null) data = data.filter((d) => !omit.includes(d));
    return (
      <Form.Field<K[]> path={`${prefix}.${fieldKey}`} label={label}>
        {(p) => (
          <Select.Multiple<K, NamedKey<K>>
            {...p}
            columns={NAMED_KEY_COLS}
            data={data}
            entryRenderKey="name"
          />
        )}
      </Form.Field>
    );
  };

const TerminalConfigField = buildNamedKeySelect(
  "Terminal Configuration",
  "terminalConfig",
  [
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
);

const AccelSensitivityUnitsField = buildNamedKeySelect<AccelSensitivityUnits>(
  "Sensitivity Units",
  "sensitivityUnits",
  [
    {
      key: "mVoltsPerG",
      name: "mV/g",
    },
    {
      key: "VoltsPerG",
      name: "V/g",
    },
  ],
);

const ExcitSourceField = buildNamedKeySelect<ExcitationSource>(
  "Excitation Source",
  "excitSource",
  [
    {
      key: "Internal",
      name: "Internal",
    },
    {
      key: "External",
      name: "External",
    },
  ],
);

const AccelerationUnitsField = buildNamedKeySelect<AccelerationUnits>(
  "Acceleration Units",
  "accelUnits",
  [
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
);

const BridgeConfigField = buildNamedKeySelect("Bridge Configuration", "bridgeConfig", [
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
]);

const ShuntResistorLocField = buildNamedKeySelect(
  "Shunt Resistor Location",
  "shuntLoc",
  [
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
);

const ResistanceConfigField = buildNamedKeySelect(
  "Resistance Configuration",
  "resConfig",
  [
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
);

const StrainConfig = buildNamedKeySelect("Strain Configuration", "strainConfig", [
  {
    key: "FullBridgeI",
    name: "Full Bridge I",
  },
  {
    key: "FullBridgeII",
    name: "Full Bridge II",
  },
  {
    key: "Full BridgeIII",
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
]);

const buildNumericField =
  (label: string, fieldKey_: string): FC<FormProps> =>
  ({ prefix: path, fieldKey = fieldKey_ }): ReactElement => (
    <Form.Field<number> path={`${path}.${fieldKey}`} label={label}>
      {(p) => <Input.Numeric {...p} />}
    </Form.Field>
  );

const MinValueField = buildNumericField("Minimum Value", "minValue");
const MaxValueField = buildNumericField("Maximum Value", "maxValue");
const SensitivityField = buildNumericField("Sensitivity", "sensitivity");

const MinMaxValueFields = ({ prefix }: FormProps): ReactElement => (
  <Align.Space direction="x">
    <MinValueField prefix={prefix} />
    <MaxValueField prefix={prefix} />
  </Align.Space>
);

const AmpsOnlyUnitsField = buildNamedKeySelect("Current Units", "units", [
  {
    key: "Amps",
    name: "Amps",
  },
]);

const ForceUnitsField = buildNamedKeySelect<ForceUnits>("Force Units", "units", [
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
]);

const ElectricalUnitsField = buildNamedKeySelect<ElectricalUnits>(
  "Electrical Units",
  "units",
  [
    {
      key: "VoltsPerVolt",
      name: "Volts per Volt",
    },
    {
      key: "mVoltsPerVolt",
      name: "mV per Volt",
    },
  ],
);

const PressureUnitsField = buildNamedKeySelect("Pressure Units", "units", [
  {
    key: "Pascals",
    name: "Pascals",
  },
  {
    key: "PSI",
    name: "PSI",
  },
]);

const TemperatureUnitsField = buildNamedKeySelect("Temperature Units", "units", [
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
]);

const TorqueUnitsField = buildNamedKeySelect<TorqueUnits>("Torque Units", "units", [
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
]);

const ANALOG_INPUT_FORMS: Record<AIChanType, FC<FormProps>> = {
  ai_accel: ({ prefix }) => (
    <>
      <ChannelField prefix={prefix} />
      <TerminalConfigField prefix={prefix} />
      <MinMaxValueFields prefix={prefix} />
      <SensitivityField prefix={prefix} />
      <AccelSensitivityUnitsField prefix={prefix} />
      <ExcitSourceField
        prefix={prefix}
        fieldKey="currExcitSource"
        label="Current Excitation Source"
      />
      <Form.NumericField
        path={`${prefix}.currExcitVal`}
        label="Current Excitation Value"
      />
    </>
  ),
  ai_accel_4_wire_dc_voltage: ({ prefix }) => (
    <>
      <ChannelField prefix={prefix} />
      <TerminalConfigField prefix={prefix} />
      <MinMaxValueFields prefix={prefix} />
      <SensitivityField prefix={prefix} />
      <AccelSensitivityUnitsField prefix={prefix} />
      <ExcitSourceField
        prefix={prefix}
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
    </>
  ),
  ai_accel_charge: ({ prefix }) => (
    <>
      <BaseAIAccelForm prefix={prefix} />
      <AccelerationUnitsField prefix={prefix} />
    </>
  ),
  ai_bridge: ({ prefix }) => (
    <>
      <ChannelField prefix={prefix} />
      <MinMaxValueFields prefix={prefix} />
      <BridgeConfigField prefix={prefix} />
      <ExcitSourceField
        prefix={prefix}
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
    </>
  ),
  ai_charge: ({ prefix }) => {
    const Units = buildNamedKeySelect("Charge Units", "units", [
      {
        key: "Coulombs",
        name: "Coulombs",
      },
      {
        key: "PicoCoulombs",
        name: "nC",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <Units prefix={prefix} />
      </>
    );
  },
  ai_current: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <AmpsOnlyUnitsField prefix={prefix} />
        <ShuntResistorLocField prefix={prefix} />
        <Form.NumericField
          path={`${prefix}.extShuntResistorVal`}
          label="Shunt Resistance"
        />
      </>
    );
  },
  ai_current_rms: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <AmpsOnlyUnitsField prefix={prefix} />
        <ShuntResistorLocField prefix={prefix} />
        <Form.NumericField
          path={`${prefix}.extShuntResistorVal`}
          label="Shunt Resistance"
        />
      </>
    );
  },
  // TODO: Add support for entering coefficients
  ai_force_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <ForceUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
        <ElectricalUnitsField prefix={prefix} />
        <ForceUnitsField
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
      </>
    );
  },
  ai_force_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <ForceUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField prefix={prefix} />
        {/* electricalVals */}
      </>
    );
  },
  ai_force_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <ForceUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <ElectricalUnitsField prefix={prefix} />
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
      </>
    );
  },
  ai_force_epe: ({ prefix }) => {
    const SensitivityUnits = buildNamedKeySelect(
      "Sensitivity Units",
      "sensitivityUnits",
      [
        {
          key: "mVoltsPerNewton",
          name: "mV/N",
        },
        {
          key: "mVoltsPerPound",
          name: "mV/lb",
        },
      ],
    );
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <ForceUnitsField prefix={prefix} omit={["KilogramForce"]} />
        <SensitivityField prefix={prefix} />
        <SensitivityUnits prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
      </>
    );
  },
  ai_freq_voltage: ({ prefix }) => {
    const UnitsField = buildNamedKeySelect("Frequency Units", "units", [
      {
        key: "Hz",
        name: "Hertz",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <UnitsField prefix={prefix} />
        <Form.NumericField path={`${prefix}.thresholdLevel`} label="Threshold Level" />
        <Form.NumericField path={`${prefix}.hysteresis`} label="Hysteresis" />
      </>
    );
  },
  ai_microphone: ({ prefix }) => {
    const UnitsField = buildNamedKeySelect("Sound Pressure Units", "units", [
      {
        key: "Pascals",
        name: "Pascals",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <UnitsField prefix={prefix} />
        <Form.NumericField
          path={`${prefix}.micSensitivity`}
          label="Microphone Sensitivity"
        />
        <Form.NumericField
          path={`${prefix}.maxSndPressLevel`}
          label="Max Sound Pressure Level"
        />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
      </>
    );
  },
  ai_pressure_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <PressureUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
        <ElectricalUnitsField prefix={prefix} />
        <PressureUnitsField
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
      </>
    );
  },
  ai_pressure_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <PressureUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField prefix={prefix} />
        {/* electricalVals */}
      </>
    );
  },
  ai_pressure_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <PressureUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <ElectricalUnitsField prefix={prefix} />
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
      </>
    );
  },
  ai_resistance_chan: ({ prefix }) => {
    const UnitsField = buildNamedKeySelect("Resistance Units", "units", [
      {
        key: "Ohms",
        name: "Ohms",
      },
    ]);

    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <UnitsField prefix={prefix} />
        <ResistanceConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
      </>
    );
  },
  ai_rosette_strain_gauge: ({ prefix }) => {
    const TypeField = buildNamedKeySelect("Rosette Type", "rosetteType", [
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
    ]);
    const MeasureTypeField = buildNamedKeySelectMultiple(
      "Measurement Type",
      "measureType",
      [
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
    );

    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
        <TypeField prefix={prefix} />
        <Form.NumericField path={`${prefix}.gaugeFactor`} label="Gauge Factor" />
        <Form.NumericField path={`${prefix}.poissonRatio`} label="Poisson Ratio" />
        <Form.NumericField
          path={`${prefix}.gageOrientation`}
          label="Gage Orientation"
        />
        <MeasureTypeField prefix={prefix} />
        <StrainConfig prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" />
        <Form.NumericField
          path={`${prefix}.nominalGageResistance`}
          label="Nominal Gage Resistance"
        />
        <Form.NumericField path={`${prefix}.poissonRatio`} label="Poisson's Ratio" />
        <Form.NumericField
          path={`${prefix}.leadWireResistance`}
          label="Lead Wire Resistance"
        />
      </>
    );
  },
  ai_rtd: ({ prefix }) => {
    const RTDTypeField = buildNamedKeySelect("RTD Type", "rtdType", [
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
    ]);

    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TemperatureUnitsField prefix={prefix} />
        <RTDTypeField prefix={prefix} />
        <ResistanceConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
      </>
    );
  },
  ai_strain_gauge: ({ prefix }) => {
    const StrainUnitsField = buildNamedKeySelect("Strain Units", "units", [
      {
        key: "Strain",
        name: "Strain",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <StrainUnitsField prefix={prefix} />
        <StrainConfig prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField path={`${prefix}.gageFactor`} label="Gage Factor" />
        <Form.NumericField
          path={`${prefix}.initialBridgeVoltage`}
          label="Initial Bridge Voltage"
        />
        <Form.NumericField
          path={`${prefix}.nominalGageResistance`}
          label="Nominal Gage Resistance"
        />
        <Form.NumericField path={`${prefix}.poissonRatio`} label="Poisson's Ratio" />
        <Form.NumericField
          path={`${prefix}.leadWireResistance`}
          label="Lead Wire Resistance"
        />
      </>
    );
  },
  ai_temp_builtin: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <TemperatureUnitsField prefix={prefix} />
      </>
    );
  },
  ai_thermocouple: ({ prefix }) => {
    const ThermocoupleTypeField = buildNamedKeySelect(
      "Thermocouple Type",
      "thermocoupleType",
      [
        {
          key: "B",
          name: "B",
        },
        {
          key: "E",
          name: "E",
        },
        {
          key: "J",
          name: "J",
        },
        {
          key: "K",
          name: "K",
        },
        {
          key: "N",
          name: "N",
        },
        {
          key: "R",
          name: "R",
        },
        {
          key: "S",
          name: "S",
        },
        {
          key: "T",
          name: "T",
        },
      ],
    );
    const CJCSourceField = buildNamedKeySelect("CJC Source", "cjcSource", [
      {
        key: "BuiltIn",
        name: "Built In",
      },
      {
        key: "ConstVal",
        name: "Constant Value",
      },
      {
        key: "Chan",
        name: "Channel",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TemperatureUnitsField prefix={prefix} />
        <ThermocoupleTypeField prefix={prefix} />
        <CJCSourceField prefix={prefix} />
        <Form.NumericField path={`${prefix}.cjcVal`} label="CJC Value" />
      </>
    );
  },
  ai_thermistor_iex: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TemperatureUnitsField prefix={prefix} />
        <ResistanceConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
        <Form.NumericField path={`${prefix}.a`} label="A" />
        <Form.NumericField path={`${prefix}.b`} label="B" />
        <Form.NumericField path={`${prefix}.c`} label="C" />
      </>
    );
  },
  ai_thermistor_vex: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TemperatureUnitsField prefix={prefix} />
        <ResistanceConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField path={`${prefix}.a`} label="A" />
        <Form.NumericField path={`${prefix}.b`} label="B" />
        <Form.NumericField path={`${prefix}.c`} label="C" />
        <ResistanceConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="voltageExcitSource"
          label="Voltage Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.voltageExcitVal`}
          label="Voltage Excitation Value"
        />
        <Form.NumericField path={`${prefix}.a`} label="A" />
        <Form.NumericField path={`${prefix}.b`} label="B" />
        <Form.NumericField path={`${prefix}.c`} label="C" />
        <Form.NumericField path={`${prefix}.r1`} label="R1" />
      </>
    );
  },
  ai_torque_bridge_polynomial: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TorqueUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
        <ElectricalUnitsField prefix={prefix} />
        <TorqueUnitsField
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
      </>
    );
  },
  ai_torque_bridge_table: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TorqueUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        {/* physicalVals */}
        <ElectricalUnitsField prefix={prefix} />
        {/* electricalVals */}
      </>
    );
  },
  ai_torque_bridge_two_point_lin: ({ prefix }) => {
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <TorqueUnitsField prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
          prefix={prefix}
          fieldKey="physicalUnits"
          label="Physical Units"
        />
        <ElectricalUnitsField prefix={prefix} />
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
      </>
    );
  },
  ai_voltage: ({ prefix }) => {
    const VelocityUnits = buildNamedKeySelect("Velocity Units", "units", [
      {
        key: "MetersPerSecond",
        name: "m/s",
      },
      {
        key: "InchesPerSecond",
        name: "in/s",
      },
    ]);
    const SensitivityUnits = buildNamedKeySelect(
      "Sensitivity Units",
      "sensitivityUnits",
      [
        {
          key: "MillivoltsPerMillimeterPerSecond",
          name: "mV/mm/s",
        },
        {
          key: "MilliVoltsPerInchPerSecond",
          name: "mV/in/s",
        },
      ],
    );
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <ElectricalUnitsField prefix={prefix} />
        <VelocityUnits prefix={prefix} />
        <Form.NumericField path={`${prefix}.sensitivity`} label="Sensitivity" />
        <SensitivityUnits prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
          fieldKey="currExcitSource"
          label="Current Excitation Source"
        />
        <Form.NumericField
          path={`${prefix}.currExcitVal`}
          label="Current Excitation Value"
        />
      </>
    );
  },
  ai_voltage_rms: ({ prefix }) => {
    const VoltageUnits = buildNamedKeySelect("Units", "units", [
      {
        key: "Volts",
        name: "Volts",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <TerminalConfigField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <VoltageUnits prefix={prefix} />
      </>
    );
  },
  ai_voltage_with_excit: ({ prefix }) => {
    const VoltageUnits = buildNamedKeySelect("Units", "units", [
      {
        key: "Volts",
        name: "Volts",
      },
    ]);
    return (
      <>
        <ChannelField prefix={prefix} />
        <MinMaxValueFields prefix={prefix} />
        <VoltageUnits prefix={prefix} />
        <BridgeConfigField prefix={prefix} />
        <ExcitSourceField
          prefix={prefix}
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
      </>
    );
  },
};
