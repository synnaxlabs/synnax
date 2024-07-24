// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC } from "react";
import { z } from "zod";

import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";
import {
  ButtonForm,
  CommonNonToggleForm,
  CommonToggleForm,
  SetpointForm,
  LightForm,
  SolenoidValveForm,
  type SymbolFormProps,
  TankForm,
  ValueForm,
} from "@/vis/schematic/Forms";
import { type LabelExtensionProps } from "@/vis/schematic/Labeled";
import {
  AngledReliefValve,
  AngledReliefValvePreview,
  AngledValve,
  AngledValvePreview,
  type AngledValveProps,
  BurstDisc,
  BurstDiscPreview,
  Button,
  ButtonPreview,
  type ButtonProps,
  Cap,
  CapPreview,
  CavityPump,
  CavityPumpPreview,
  type CavityPumpProps,
  CheckValve,
  CheckValvePreview,
  type CheckValveProps,
  ElectricRegulator,
  ElectricRegulatorPreview,
  type ElectricRegulatorProps,
  Filter,
  FilterPreview,
  type FilterProps,
  FourWayValve,
  FourWayValvePreview,
  InputPreview,
  type InputProps,
  Light,
  LightPreview,
  type LightProps,
  ManualValve,
  ManualValvePreview,
  type ManualValveProps,
  NeedleValve,
  NeedleValvePreview,
  type NeedleValveProps,
  Orifice,
  OrificePreview,
  type OrificeProps,
  PistonPump,
  PistonPumpPreview,
  type PistonPumpProps,
  Pump,
  PumpPreview,
  type PumpProps,
  Regulator,
  RegulatorPreview,
  type RegulatorProps,
  ReliefValve,
  ReliefValvePreview,
  type ReliefValveProps,
  RotaryMixer,
  RotaryMixerPreview,
  type RotaryMixerProps,
  ScrewPump,
  ScrewPumpPreview,
  type ScrewPumpProps,
  SetPoint,
  SolenoidValve,
  SolenoidValvePreview,
  type SolenoidValveProps,
  StaticMixer,
  StaticMixerPreview,
  type StaticMixerProps,
  Switch,
  SwitchPreview,
  type SwitchProps,
  type SymbolProps,
  Tank,
  TankPreview,
  type TankProps,
  ThreeWayValve,
  ThreeWayValvePreview,
  type ThreeWayValveProps,
  VacuumPump,
  VacuumPumpPreview,
  type VacuumPumpProps,
  Value,
  ValuePreview,
  type ValueProps,
  Valve,
  ValvePreview,
  type ValveProps,
} from "@/vis/schematic/Symbols";

export interface Spec<P extends object> {
  key: Variant;
  name: string;
  Form: FC<SymbolFormProps>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<SymbolProps<P>>;
  zIndex: number;
}

const Z_INDEX_UPPER = 4;
const Z_INDEX_LOWER = 2;

const VARIANTS = [
  "angledReliefValve",
  "angledValve",
  "burstDisc",
  "button",
  "cap",
  "cavityPump",
  "checkValve",
  "electricRegulator",
  "filter",
  "fourWayValve",
  "setpoint",
  "light",
  "manualValve",
  "needleValve",
  "orifice",
  "pistonPump",
  "pump",
  "regulator",
  "reliefValve",
  "rotaryMixer",
  "screwPump",
  "solenoidValve",
  "staticMixer",
  "switch",
  "tank",
  "threeWayValve",
  "vacuumPump",
  "value",
  "valve",
] as const;

export const typeZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof typeZ>;

const ZERO_PROPS = {
  orientation: "left" as const,
};

const ZERO_NUMERIC_STRINGER_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("string", {
    connections: [
      {
        from: "valueStream",
        to: "rollingAverage",
      },
      {
        from: "rollingAverage",
        to: "stringifier",
      },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2 }),
    },
    outlet: "stringifier",
  }),
};

const ZERO_NUMERIC_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("number", {
    connections: [],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
    },
    outlet: "valueStream",
  }),
};

const ZERO_NUMERIC_SINK_PROPS = {
  ...ZERO_PROPS,
  sink: telem.sinkPipeline("number", {
    connections: [],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
    },
    inlet: "setter",
  }),
};

const ZERO_BOOLEAN_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("boolean", {
    connections: [
      {
        from: "valueStream",
        to: "threshold",
      },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
    },
    outlet: "threshold",
  }),
};

const ZERO_BOOLEAN_SINK_PROPS = {
  ...ZERO_PROPS,
  sink: telem.sinkPipeline("boolean", {
    connections: [
      {
        from: "setpoint",
        to: "setter",
      },
    ],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  }),
};

const ZERO_TOGGLE_PROPS = {
  ...ZERO_BOOLEAN_SOURCE_PROPS,
  ...ZERO_BOOLEAN_SINK_PROPS,
};

type zeroLabelReturn = {
  label: LabelExtensionProps;
};

const zeroLabel = (label: string): zeroLabelReturn => ({
  label: {
    label,
    level: "p",
    orientation: "top",
  },
});

const threeWayValve: Spec<ThreeWayValveProps> = {
  name: "Three Way Valve",
  key: "threeWayValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Three Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: ThreeWayValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const valve: Spec<ValveProps> = {
  name: "Valve",
  key: "valve",
  Form: CommonToggleForm,
  Symbol: Valve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: ValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const solenoidValve: Spec<SolenoidValveProps> = {
  name: "Solenoid Valve",
  key: "solenoidValve",
  Form: SolenoidValveForm,
  Symbol: SolenoidValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Solenoid Valve"),
    normallyOpen: false,
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: SolenoidValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const fourWayValve: Spec<ValveProps> = {
  name: "Four Way Valve",
  key: "fourWayValve",
  Form: CommonToggleForm,
  Symbol: FourWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Four Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: FourWayValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const angledValve: Spec<AngledValveProps> = {
  name: "Angled Valve",
  key: "angledValve",
  Form: CommonToggleForm,
  Symbol: AngledValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: AngledValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const pump: Spec<PumpProps> = {
  name: "Pump",
  key: "pump",
  Form: CommonToggleForm,
  Symbol: Pump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: PumpPreview,
  zIndex: Z_INDEX_UPPER,
};

const screwPump: Spec<ScrewPumpProps> = {
  name: "Screw Pump",
  key: "screwPump",
  Form: CommonToggleForm,
  Symbol: ScrewPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Screw Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: ScrewPumpPreview,
  zIndex: Z_INDEX_UPPER,
};

const tank: Spec<TankProps> = {
  name: "Tank",
  key: "tank",
  Form: TankForm,
  Symbol: Tank,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Tank"),
    dimensions: {
      width: 100,
      height: 200,
    },
    ...ZERO_PROPS,
  }),
  Preview: TankPreview,
  zIndex: Z_INDEX_LOWER,
};

const reliefValve: Spec<ReliefValveProps> = {
  name: "Relief Valve",
  key: "reliefValve",
  Form: CommonNonToggleForm,
  Symbol: ReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Relief Valve"),
    ...ZERO_PROPS,
  }),
  Preview: ReliefValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const regulator: Spec<RegulatorProps> = {
  name: "Regulator",
  key: "regulator",
  Form: CommonNonToggleForm,
  Symbol: Regulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: RegulatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const electricRegulator: Spec<ElectricRegulatorProps> = {
  name: "Electric Regulator",
  key: "electricRegulator",
  Form: CommonNonToggleForm,
  Symbol: ElectricRegulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Electric Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: ElectricRegulatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const burstDisc: Spec<ReliefValveProps> = {
  name: "Burst Disc",
  key: "burstDisc",
  Form: CommonNonToggleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: BurstDiscPreview,
  zIndex: Z_INDEX_UPPER,
};

const cap: Spec<ReliefValveProps> = {
  name: "Cap",
  key: "cap",
  Form: CommonNonToggleForm,
  Symbol: Cap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cap"),
    ...ZERO_PROPS,
  }),
  Preview: CapPreview,
  zIndex: Z_INDEX_UPPER,
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual Valve",
  key: "manualValve",
  Form: CommonNonToggleForm,
  Symbol: ManualValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Manual Valve"),
    ...ZERO_PROPS,
  }),
  Preview: ManualValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const filter: Spec<FilterProps> = {
  name: "Filter",
  key: "filter",
  Form: CommonNonToggleForm,
  Symbol: Filter,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Filter"),
    ...ZERO_PROPS,
  }),
  Preview: FilterPreview,
  zIndex: Z_INDEX_UPPER,
};

const needleValve: Spec<NeedleValveProps> = {
  name: "Needle Valve",
  key: "needleValve",
  Form: CommonNonToggleForm,
  Symbol: NeedleValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Needle Valve"),
    ...ZERO_PROPS,
  }),
  Preview: NeedleValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const checkValve: Spec<CheckValveProps> = {
  name: "Check Valve",
  key: "checkValve",
  Form: CommonNonToggleForm,
  Symbol: CheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: CheckValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const orifice: Spec<OrificeProps> = {
  name: "Orifice",
  key: "orifice",
  Form: CommonNonToggleForm,
  Symbol: Orifice,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Orifice"),
    ...ZERO_PROPS,
  }),
  Preview: OrificePreview,
  zIndex: Z_INDEX_UPPER,
};

const angledReliefValve: Spec<ReliefValveProps> = {
  name: "Angled Relief Valve",
  key: "angledReliefValve",
  Form: CommonNonToggleForm,
  Symbol: AngledReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Relief Valve"),
    ...ZERO_PROPS,
  }),
  Preview: AngledReliefValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const value: Spec<ValueProps> = {
  name: "Value",
  key: "value",
  Form: ValueForm,
  Symbol: Value,
  Preview: ValuePreview,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    units: "psi",
    level: "h5",
    ...zeroLabel("Value"),
    ...ZERO_PROPS,
    telem: ZERO_NUMERIC_STRINGER_SOURCE_PROPS.source,
  }),
  zIndex: Z_INDEX_UPPER,
};

const button: Spec<ButtonProps> = {
  name: "Button",
  key: "button",
  Symbol: Button,
  Form: ButtonForm,
  Preview: ButtonPreview,
  defaultProps: () => ({
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
  }),
  zIndex: Z_INDEX_UPPER,
};

const switch_: Spec<SwitchProps> = {
  name: "Switch",
  key: "switch",
  Symbol: Switch,
  Preview: SwitchPreview,
  Form: CommonToggleForm,
  defaultProps: () => ({
    ...zeroLabel("Switch"),
    ...ZERO_TOGGLE_PROPS,
  }),
  zIndex: Z_INDEX_UPPER,
};

const vacuumPump: Spec<VacuumPumpProps> = {
  name: "Vacuum Pump",
  key: "vacuumPump",
  Symbol: VacuumPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Vacuum Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: VacuumPumpPreview,
  zIndex: Z_INDEX_UPPER,
};

const cavityPump: Spec<CavityPumpProps> = {
  name: "Cavity Pump",
  key: "cavityPump",
  Symbol: CavityPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cavity Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: CavityPumpPreview,
  zIndex: Z_INDEX_UPPER,
};

const pistonPump: Spec<PistonPumpProps> = {
  name: "Piston Pump",
  key: "pistonPump",
  Symbol: PistonPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Piston Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: PistonPumpPreview,
  zIndex: Z_INDEX_UPPER,
};

const staticMixer: Spec<StaticMixerProps> = {
  name: "Static Mixer",
  key: "staticMixer",
  Symbol: StaticMixer,
  Form: CommonNonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Static Mixer"),
    ...ZERO_PROPS,
  }),
  Preview: StaticMixerPreview,
  zIndex: Z_INDEX_UPPER,
};

const rotaryMixer: Spec<RotaryMixerProps> = {
  name: "Rotary Mixer",
  key: "rotaryMixer",
  Symbol: RotaryMixer,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Rotary Mixer"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: RotaryMixerPreview,
  zIndex: Z_INDEX_UPPER,
};

const light: Spec<LightProps> = {
  name: "Light",
  key: "light",
  Symbol: Light,
  Form: LightForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Light"),
    ...ZERO_BOOLEAN_SOURCE_PROPS,
  }),
  Preview: LightPreview,
  zIndex: Z_INDEX_UPPER,
};

const setpoint: Spec<InputProps> = {
  name: "Setpoint",
  key: "setpoint",
  Symbol: SetPoint,
  Form: SetpointForm,
  defaultProps: () => ({
    units: "mV",
    ...zeroLabel("Input"),
    ...ZERO_NUMERIC_SOURCE_PROPS,
    ...ZERO_NUMERIC_SINK_PROPS,
  }),
  Preview: InputPreview,
  zIndex: Z_INDEX_UPPER,
};

export const SYMBOLS: Record<Variant, Spec<any>> = {
  value,
  light,
  switch: switch_,
  button,
  setpoint,
  tank,
  valve,
  solenoidValve,
  threeWayValve,
  fourWayValve,
  angledValve,
  manualValve,
  needleValve,
  reliefValve,
  angledReliefValve,
  checkValve,
  regulator,
  electricRegulator,
  pump,
  pistonPump,
  screwPump,
  cavityPump,
  vacuumPump,
  staticMixer,
  rotaryMixer,
  burstDisc,
  cap,
  filter,
  orifice,
};
