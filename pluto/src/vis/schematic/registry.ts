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
  CommonStyleForm,
  CommonToggleForm,
  CylinderForm,
  LightForm,
  OffPageReferenceForm,
  SetpointForm,
  type SymbolFormProps,
  TankForm,
  TextBoxForm,
  ValueForm,
} from "@/vis/schematic/Forms";
import { type LabelExtensionProps } from "@/vis/schematic/Labeled";
import { DEFAULT_BORDER_RADIUS } from "@/vis/schematic/primitives/Primitives";
import {
  Agitator,
  AgitatorPreview,
  type AgitatorProps,
  AngledReliefValve,
  AngledReliefValvePreview,
  AngledSpringLoadedReliefValve,
  AngledSpringLoadedReliefValvePreview,
  type AngledSpringLoadedReliefValveProps,
  AngledValve,
  AngledValvePreview,
  type AngledValveProps,
  Box,
  BoxPreview,
  type BoxProps,
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
  Compressor,
  CompressorPreview,
  type CompressorProps,
  CrossBeamAgitator,
  CrossBeamAgitatorPreview,
  type CrossBeamAgitatorProps,
  Cylinder,
  CylinderPreview,
  type CylinderProps,
  ElectricRegulator,
  ElectricRegulatorPreview,
  type ElectricRegulatorProps,
  Filter,
  FilterPreview,
  type FilterProps,
  FlatBladeAgitator,
  FlatBladeAgitatorPreview,
  type FlatBladeAgitatorProps,
  FourWayValve,
  FourWayValvePreview,
  HelicalAgitator,
  HelicalAgitatorPreview,
  type HelicalAgitatorProps,
  ISOBurstDisc,
  ISOBurstDiscPreview,
  ISOCap,
  ISOCapPreview,
  ISOCheckValve,
  ISOCheckValvePreview,
  type ISOCheckValveProps,
  ISOFilter,
  ISOFilterPreview,
  Light,
  LightPreview,
  type LightProps,
  ManualValve,
  ManualValvePreview,
  type ManualValveProps,
  NeedleValve,
  NeedleValvePreview,
  type NeedleValveProps,
  OffPageReference,
  OffPageReferencePreview,
  type OffPageReferenceProps,
  Orifice,
  OrificePlate,
  OrificePlatePreview,
  type OrificePlateProps,
  OrificePreview,
  type OrificeProps,
  PaddleAgitator,
  PaddleAgitatorPreview,
  type PaddleAgitatorProps,
  PistonPump,
  PistonPumpPreview,
  type PistonPumpProps,
  PropellerAgitator,
  PropellerAgitatorPreview,
  type PropellerAgitatorProps,
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
  Setpoint,
  SetpointPreview,
  type SetpointProps,
  SolenoidValve,
  SolenoidValvePreview,
  type SolenoidValveProps,
  SpringLoadedReliefValve,
  SpringLoadedReliefValvePreview,
  type SpringLoadedReliefValveProps,
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
  TextBox,
  TextBoxPreview,
  type TextBoxProps,
  ThreeWayValve,
  ThreeWayValvePreview,
  type ThreeWayValveProps,
  TJunction,
  TJunctionPreview,
  type TJunctionProps,
  VacuumPump,
  VacuumPumpPreview,
  type VacuumPumpProps,
  Value,
  ValuePreview,
  type ValueProps,
  Valve,
  ValvePreview,
  type ValveProps,
  Vent,
  VentPreview,
  type VentProps,
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
  "agitator",
  "angledReliefValve",
  "angledSpringLoadedReliefValve",
  "angledValve",
  "offPageReference",
  "box",
  "burstDisc",
  "isoBurstDisc",
  "button",
  "cap",
  "cavityPump",
  "checkValve",
  "cylinder",
  "compressor",
  "crossBeamAgitator",
  "electricRegulator",
  "filter",
  "flatBladeAgitator",
  "fourWayValve",
  "helicalAgitator",
  "isoCap",
  "isoCheckValve",
  "isoFilter",
  "light",
  "manualValve",
  "needleValve",
  "orifice",
  "orificePlate",
  "paddleAgitator",
  "propellerAgitator",
  "pistonPump",
  "pump",
  "regulator",
  "reliefValve",
  "rotaryMixer",
  "screwPump",
  "setpoint",
  "solenoidValve",
  "springLoadedReliefValve",
  "staticMixer",
  "switch",
  "tank",
  "textBox",
  "threeWayValve",
  "vacuumPump",
  "value",
  "valve",
  "vent",
  "tJunction",
] as const;

export const typeZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof typeZ>;

const ZERO_PROPS = { orientation: "left" as const, scale: 1 };
const ZERO_NUMERIC_STRINGER_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("string", {
    connections: [
      { from: "valueStream", to: "rollingAverage" },
      { from: "rollingAverage", to: "stringifier" },
    ],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      stringifier: telem.stringifyNumber({ precision: 2, notation: "standard" }),
    },
    outlet: "stringifier",
  }),
};

const ZERO_NUMERIC_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("number", {
    connections: [],
    segments: { valueStream: telem.streamChannelValue({ channel: 0 }) },
    outlet: "valueStream",
  }),
};

const ZERO_NUMERIC_SINK_PROPS = {
  ...ZERO_PROPS,
  sink: telem.sinkPipeline("number", {
    connections: [],
    segments: { setter: control.setChannelValue({ channel: 0 }) },
    inlet: "setter",
  }),
};

const ZERO_BOOLEAN_SOURCE_PROPS = {
  ...ZERO_PROPS,
  source: telem.sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: telem.streamChannelValue({ channel: 0 }),
      threshold: telem.withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
    },
    outlet: "threshold",
  }),
};

const ZERO_BOOLEAN_SINK_PROPS = {
  ...ZERO_PROPS,
  control: { show: true },
  sink: telem.sinkPipeline("boolean", {
    connections: [{ from: "setpoint", to: "setter" }],
    segments: {
      setter: control.setChannelValue({ channel: 0 }),
      setpoint: telem.setpoint({ truthy: 1, falsy: 0 }),
    },
    inlet: "setpoint",
  }),
};

const ZERO_TOGGLE_PROPS = { ...ZERO_BOOLEAN_SOURCE_PROPS, ...ZERO_BOOLEAN_SINK_PROPS };

type zeroLabelReturn = { label: LabelExtensionProps };

const zeroLabel = (label: string): zeroLabelReturn => ({
  label: { label, level: "p", orientation: "top", maxInlineSize: 150, align: "center" },
});

const ZERO_DIMENSIONS = { width: 125, height: 200 };

const ZERO_BOX_PROPS = { dimensions: ZERO_DIMENSIONS };

const ZERO_BOX_BORDER_RADIUS = 3;

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
  Form: CommonToggleForm,
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
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("Tank"),
    borderRadius: DEFAULT_BORDER_RADIUS,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: TankPreview,
  zIndex: Z_INDEX_LOWER,
};

const cylinder: Spec<CylinderProps> = {
  name: "Cylinder",
  key: "cylinder",
  Form: CylinderForm,
  Symbol: Cylinder,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("cylinder"),
    dimensions: {
      width: 66,
      height: 181,
    },
    ...ZERO_PROPS,
  }),
  Preview: CylinderPreview,
  zIndex: Z_INDEX_LOWER,
};

const box: Spec<BoxProps> = {
  name: "Box",
  key: "box",
  Form: () => TankForm({ includeBorderRadius: true }),
  Symbol: Box,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    backgroundColor: t.colors.gray.l1.setAlpha(0).rgba255,
    ...zeroLabel("Box"),
    borderRadius: ZERO_BOX_BORDER_RADIUS,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: BoxPreview,
  zIndex: Z_INDEX_LOWER,
};

const reliefValve: Spec<ReliefValveProps> = {
  name: "Relief Valve",
  key: "reliefValve",
  Form: CommonStyleForm,
  Symbol: ReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Relief Valve"),
    ...ZERO_PROPS,
  }),
  Preview: ReliefValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const springLoadedReliefValve: Spec<SpringLoadedReliefValveProps> = {
  name: "Spring Loaded Relief Valve",
  key: "springLoadedReliefValve",
  Form: CommonStyleForm,
  Symbol: SpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Spring Loaded Relief Valve"),
    ...ZERO_PROPS,
  }),
  Preview: SpringLoadedReliefValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const angledSpringLoadedReliefValve: Spec<AngledSpringLoadedReliefValveProps> = {
  name: "Angled Spring Loaded Relief Valve",
  key: "angledSpringLoadedReliefValve",
  Form: CommonStyleForm,
  Symbol: AngledSpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Spring Loaded Relief Valve"),
    ...ZERO_PROPS,
  }),
  Preview: AngledSpringLoadedReliefValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const regulator: Spec<RegulatorProps> = {
  name: "Regulator",
  key: "regulator",
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: BurstDiscPreview,
  zIndex: Z_INDEX_UPPER,
};

const isoBurstDisc: Spec<ReliefValveProps> = {
  name: "ISO Burst Disc",
  key: "isoBurstDisc",
  Form: CommonStyleForm,
  Symbol: ISOBurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: ISOBurstDiscPreview,
  zIndex: Z_INDEX_UPPER,
};

const cap: Spec<ReliefValveProps> = {
  name: "Cap",
  key: "cap",
  Form: CommonStyleForm,
  Symbol: Cap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cap"),
    ...ZERO_PROPS,
  }),
  Preview: CapPreview,
  zIndex: Z_INDEX_UPPER,
};

const isoCap: Spec<ReliefValveProps> = {
  name: "ISO Cap",
  key: "isoCap",
  Form: CommonStyleForm,
  Symbol: ISOCap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Cap"),
    ...ZERO_PROPS,
  }),
  Preview: ISOCapPreview,
  zIndex: Z_INDEX_UPPER,
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual Valve",
  key: "manualValve",
  Form: CommonStyleForm,
  Symbol: ManualValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Manual Valve"),
    ...ZERO_PROPS,
  }),
  Preview: ManualValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const orificePlate: Spec<OrificePlateProps> = {
  name: "Orifice Plate",
  key: "orificePlate",
  Form: CommonStyleForm,
  Symbol: OrificePlate,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Orifice Plate"),
    ...ZERO_PROPS,
  }),
  Preview: OrificePlatePreview,
  zIndex: Z_INDEX_UPPER,
};

const isoFilter: Spec<ManualValveProps> = {
  name: "ISO Filter",
  key: "isoFilter",
  Form: CommonStyleForm,
  Symbol: ISOFilter,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Filter"),
    ...ZERO_PROPS,
  }),
  Preview: ISOFilterPreview,
  zIndex: Z_INDEX_UPPER,
};

const filter: Spec<FilterProps> = {
  name: "Filter",
  key: "filter",
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
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
  Form: CommonStyleForm,
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
    inlineSize: 70,
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
  defaultProps: (t) => ({
    color: t.colors.primary.z.rgba255,
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    onClickDelay: 0,
    scale: null,
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
    scale: null,
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
  Form: CommonStyleForm,
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

const setpoint: Spec<SetpointProps> = {
  name: "Setpoint",
  key: "setpoint",
  Symbol: Setpoint,
  Form: SetpointForm,
  defaultProps: (t) => ({
    units: "mV",
    color: t.colors.gray.l4.rgba255,
    ...zeroLabel("Setpoint"),
    ...ZERO_NUMERIC_SOURCE_PROPS,
    ...ZERO_NUMERIC_SINK_PROPS,
  }),
  Preview: SetpointPreview,
  zIndex: Z_INDEX_UPPER,
};

const agitator: Spec<AgitatorProps> = {
  name: "Agitator",
  key: "agitator",
  Symbol: Agitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: AgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const propellerAgitator: Spec<PropellerAgitatorProps> = {
  name: "Propeller Agitator",
  key: "propellerAgitator",
  Symbol: PropellerAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Propeller Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: PropellerAgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const flatBladeAgitator: Spec<FlatBladeAgitatorProps> = {
  name: "Flat Blade Agitator",
  key: "flatBladeAgitator",
  Symbol: FlatBladeAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Flat Blade Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: FlatBladeAgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const paddleAgitator: Spec<PaddleAgitatorProps> = {
  name: "Paddle Agitator",
  key: "paddleAgitator",
  Symbol: PaddleAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Paddle Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: PaddleAgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const crossBeamAgitator: Spec<CrossBeamAgitatorProps> = {
  name: "Cross Beam Agitator",
  key: "crossBeamAgitator",
  Symbol: CrossBeamAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cross Beam Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: CrossBeamAgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const helicalAgitator: Spec<HelicalAgitatorProps> = {
  name: "Helical Agitator",
  key: "helicalAgitator",
  Symbol: HelicalAgitator,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Helical Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: HelicalAgitatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const compressor: Spec<CompressorProps> = {
  name: "Compressor",
  key: "compressor",
  Symbol: Compressor,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: CompressorPreview,
  zIndex: Z_INDEX_UPPER,
};

const textBox: Spec<TextBoxProps> = {
  name: "Text Box",
  key: "textBox",
  Symbol: TextBox,
  Form: TextBoxForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    autoFit: true,
    align: "center",
    ...zeroLabel("Text Box"),
    ...ZERO_PROPS,
    ...ZERO_BOX_PROPS,
    level: "p",
    text: "Text Box",
    width: 75,
  }),
  Preview: TextBoxPreview,
  zIndex: Z_INDEX_UPPER,
};

const offPageReference: Spec<OffPageReferenceProps> = {
  name: "Off Page Reference",
  key: "offPageReference",
  Form: OffPageReferenceForm,
  Symbol: OffPageReference,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    orientation: "right",
    ...zeroLabel("Off Page Reference"),
  }),
  Preview: OffPageReferencePreview,
  zIndex: Z_INDEX_UPPER,
};

const isoCheckValve: Spec<ISOCheckValveProps> = {
  name: "ISO Check Valve",
  key: "isoCheckValve",
  Form: CommonStyleForm,
  Symbol: ISOCheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: ISOCheckValvePreview,
  zIndex: Z_INDEX_UPPER,
};

const vent: Spec<VentProps> = {
  name: "Vent",
  key: "vent",
  Form: CommonStyleForm,
  Symbol: Vent,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Vent"),
    ...ZERO_PROPS,
  }),
  Preview: VentPreview,
  zIndex: Z_INDEX_UPPER,
};

const tJunction: Spec<TJunctionProps> = {
  name: "T Junction",
  key: "tJunction",
  Form: CommonStyleForm,
  Symbol: TJunction,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("T Junction"),
    ...ZERO_PROPS,
  }),
  Preview: TJunctionPreview,
  zIndex: Z_INDEX_UPPER + 20,
};

export const SYMBOLS: Record<Variant, Spec<any>> = {
  value,
  button,
  tank,
  tJunction,
  switch: switch_,
  offPageReference,
  light,
  setpoint,
  box,
  textBox,
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
  isoBurstDisc,
  cap,
  isoCap,
  filter,
  isoFilter,
  orifice,
  orificePlate,
  agitator,
  propellerAgitator,
  flatBladeAgitator,
  paddleAgitator,
  crossBeamAgitator,
  helicalAgitator,
  compressor,
  isoCheckValve,
  vent,
  cylinder,
  springLoadedReliefValve,
  angledSpringLoadedReliefValve,
};
