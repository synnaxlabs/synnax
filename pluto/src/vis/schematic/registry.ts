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
  BoxForm,
  ButtonForm,
  CommonDummyToggleForm,
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
import { Primitives } from "@/vis/schematic/primitives";
import {
  type CylinderProps,
  DEFAULT_BORDER_RADIUS,
  TextBox,
  type TextBoxProps,
} from "@/vis/schematic/primitives/Primitives";
import {
  Agitator,
  type AgitatorProps,
  AngledReliefValve,
  AngledSpringLoadedReliefValve,
  type AngledSpringLoadedReliefValveProps,
  AngledValve,
  type AngledValveProps,
  Box,
  BoxPreview,
  type BoxProps,
  BurstDisc,
  type BurstDiscProps,
  Button,
  ButtonPreview,
  type ButtonProps,
  Cap,
  type CapProps,
  CavityPump,
  type CavityPumpProps,
  CheckValve,
  type CheckValveProps,
  Compressor,
  type CompressorProps,
  CrossBeamAgitator,
  type CrossBeamAgitatorProps,
  Cylinder,
  CylinderPreview,
  ElectricRegulator,
  type ElectricRegulatorProps,
  Filter,
  type FilterProps,
  FlatBladeAgitator,
  type FlatBladeAgitatorProps,
  FourWayValve,
  type FourWayValveProps,
  HelicalAgitator,
  type HelicalAgitatorProps,
  ISOBurstDisc,
  type ISOBurstDiscProps,
  ISOCap,
  type ISOCapProps,
  ISOCheckValve,
  type ISOCheckValveProps,
  ISOFilter,
  type ISOFilterProps,
  type LabelExtensionProps,
  Light,
  type LightProps,
  ManualValve,
  type ManualValveProps,
  NeedleValve,
  type NeedleValveProps,
  OffPageReference,
  OffPageReferencePreview,
  type OffPageReferenceProps,
  Orifice,
  OrificePlate,
  type OrificePlateProps,
  type OrificeProps,
  PaddleAgitator,
  type PaddleAgitatorProps,
  PistonPump,
  type PistonPumpProps,
  PropellerAgitator,
  type PropellerAgitatorProps,
  Pump,
  type PumpProps,
  Regulator,
  type RegulatorProps,
  ReliefValve,
  type ReliefValveProps,
  RotaryMixer,
  type RotaryMixerProps,
  ScrewPump,
  type ScrewPumpProps,
  Setpoint,
  SetpointPreview,
  type SetpointProps,
  SolenoidValve,
  type SolenoidValveProps,
  SpringLoadedReliefValve,
  type SpringLoadedReliefValveProps,
  StaticMixer,
  type StaticMixerProps,
  Switch,
  type SwitchProps,
  type SymbolProps,
  Tank,
  TankPreview,
  type TankProps,
  TextBoxPreview,
  ThreeWayValve,
  type ThreeWayValveProps,
  TJunction,
  type TJunctionProps,
  VacuumPump,
  type VacuumPumpProps,
  Value,
  ValuePreview,
  type ValueProps,
  Valve,
  type ValveProps,
  Vent,
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

export const variantZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof variantZ>;

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

const ZERO_DUMMY_TOGGLE_PROPS = { ...ZERO_PROPS, enabled: false, clickable: false };

type zeroLabelReturn = { label: LabelExtensionProps };

const zeroLabel = (label: string): zeroLabelReturn => ({
  label: {
    label,
    level: "p",
    orientation: "top",
    maxInlineSize: 150,
    align: "center",
    direction: "x",
  },
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
  Preview: Primitives.ThreeWayValve,
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
  Preview: Primitives.Valve,
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
  Preview: Primitives.SolenoidValve,
  zIndex: Z_INDEX_UPPER,
};

const fourWayValve: Spec<FourWayValveProps> = {
  name: "Four Way Valve",
  key: "fourWayValve",
  Form: CommonToggleForm,
  Symbol: FourWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Four Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.FourWayValve,
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
  Preview: Primitives.AngledValve,
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
  Preview: Primitives.Pump,
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
  Preview: Primitives.ScrewPump,
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
  Form: BoxForm,
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
  Form: CommonDummyToggleForm,
  Symbol: ReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.ReliefValve,
  zIndex: Z_INDEX_UPPER,
};

const springLoadedReliefValve: Spec<SpringLoadedReliefValveProps> = {
  name: "Spring Loaded Relief Valve",
  key: "springLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: SpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.SpringLoadedReliefValve,
  zIndex: Z_INDEX_UPPER,
};

const angledSpringLoadedReliefValve: Spec<AngledSpringLoadedReliefValveProps> = {
  name: "Angled Spring Loaded Relief Valve",
  key: "angledSpringLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledSpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.AngledSpringLoadedReliefValve,
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
  Preview: Primitives.Regulator,
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
  Preview: Primitives.ElectricRegulator,
  zIndex: Z_INDEX_UPPER,
};

const burstDisc: Spec<BurstDiscProps> = {
  name: "Burst Disc",
  key: "burstDisc",
  Form: CommonStyleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.BurstDisc,
  zIndex: Z_INDEX_UPPER,
};

const isoBurstDisc: Spec<ISOBurstDiscProps> = {
  name: "ISO Burst Disc",
  key: "isoBurstDisc",
  Form: CommonStyleForm,
  Symbol: ISOBurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOBurstDisc,
  zIndex: Z_INDEX_UPPER,
};

const cap: Spec<CapProps> = {
  name: "Cap",
  key: "cap",
  Form: CommonStyleForm,
  Symbol: Cap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Cap"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Cap,
  zIndex: Z_INDEX_UPPER,
};

const isoCap: Spec<ISOCapProps> = {
  name: "ISO Cap",
  key: "isoCap",
  Form: CommonStyleForm,
  Symbol: ISOCap,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Cap"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOCap,
  zIndex: Z_INDEX_UPPER,
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual Valve",
  key: "manualValve",
  Form: CommonDummyToggleForm,
  Symbol: ManualValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Manual Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.ManualValve,
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
  Preview: Primitives.OrificePlate,
  zIndex: Z_INDEX_UPPER,
};

const isoFilter: Spec<ISOFilterProps> = {
  name: "ISO Filter",
  key: "isoFilter",
  Form: CommonStyleForm,
  Symbol: ISOFilter,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("ISO Filter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOFilter,
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
  Preview: Primitives.Filter,
  zIndex: Z_INDEX_UPPER,
};

const needleValve: Spec<NeedleValveProps> = {
  name: "Needle Valve",
  key: "needleValve",
  Form: CommonDummyToggleForm,
  Symbol: NeedleValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Needle Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.NeedleValve,
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
  Preview: Primitives.CheckValve,
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
  Preview: Primitives.Orifice,
  zIndex: Z_INDEX_UPPER,
};

const angledReliefValve: Spec<ReliefValveProps> = {
  name: "Angled Relief Valve",
  key: "angledReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel("Angled Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: Primitives.AngledReliefValve,
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
  Form: CommonToggleForm,
  defaultProps: () => ({
    ...zeroLabel("Switch"),
    ...ZERO_TOGGLE_PROPS,
    scale: null,
  }),
  Preview: Primitives.Switch,
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
  Preview: Primitives.VacuumPump,
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
  Preview: Primitives.CavityPump,
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
  Preview: Primitives.PistonPump,
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
  Preview: Primitives.StaticMixer,
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
  Preview: Primitives.RotaryMixer,
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
  Preview: Primitives.Light,
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
    size: "small",
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
  Preview: Primitives.Agitator,
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
  Preview: Primitives.PropellerAgitator,
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
  Preview: Primitives.FlatBladeAgitator,
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
  Preview: Primitives.PaddleAgitator,
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
  Preview: Primitives.CrossBeamAgitator,
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
  Preview: Primitives.HelicalAgitator,
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
  Preview: Primitives.Compressor,
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
  Preview: Primitives.ISOCheckValve,
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
  Preview: Primitives.Vent,
  zIndex: Z_INDEX_UPPER,
};

const tJunction: Spec<TJunctionProps> = {
  name: "T Junction",
  key: "tJunction",
  Form: CommonStyleForm,
  Symbol: TJunction,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    ...zeroLabel(""),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.TJunction,
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
