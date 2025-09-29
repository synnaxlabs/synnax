// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group } from "@synnaxlabs/client";
import { bounds, color } from "@synnaxlabs/x";
import { type FC } from "react";
import { z } from "zod";

import { removeProps } from "@/component/removeProps";
import { Icon } from "@/icon";
import {
  BoxForm,
  ButtonForm,
  CircleForm,
  CommonDummyToggleForm,
  CommonPolygonForm,
  CommonStyleForm,
  CommonToggleForm,
  CylinderForm,
  GaugeForm,
  LightForm,
  OffPageReferenceForm,
  SetpointForm,
  SwitchForm,
  type SymbolFormProps,
  TankForm,
  TextBoxForm,
  ValueForm,
} from "@/schematic/symbol/Forms";
import {
  type CylinderProps,
  DEFAULT_BORDER_RADIUS,
  DEFAULT_POLYGON_SIDE_LENGTH,
} from "@/schematic/symbol/Primitives";
import * as Primitives from "@/schematic/symbol/Primitives";
import {
  Agitator,
  type AgitatorProps,
  AngledReliefValve,
  AngledSpringLoadedReliefValve,
  type AngledSpringLoadedReliefValveProps,
  AngledValve,
  type AngledValveProps,
  BallValve,
  type BallValveProps,
  Box,
  BoxPreview,
  type BoxProps,
  BreatherValve,
  type BreatherValveProps,
  BurstDisc,
  type BurstDiscProps,
  ButterflyValveOne,
  type ButterflyValveOneProps,
  ButterflyValveTwo,
  type ButterflyValveTwoProps,
  Button,
  ButtonPreview,
  type ButtonProps,
  Cap,
  type CapProps,
  CavityPump,
  type CavityPumpProps,
  CentrifugalCompressor,
  type CentrifugalCompressorProps,
  CheckValve,
  type CheckValveProps,
  CheckValveWithArrow,
  type CheckValveWithArrowProps,
  Circle,
  Compressor,
  type CompressorProps,
  CrossBeamAgitator,
  type CrossBeamAgitatorProps,
  CrossJunction,
  type CrossJunctionProps,
  CustomStatic,
  type CustomStaticProps,
  Cylinder,
  CylinderPreview,
  DiaphragmPump,
  type DiaphragmPumpProps,
  EjectionPump,
  type EjectionPumpProps,
  EjectorCompressor,
  type EjectorCompressorProps,
  ElectricRegulator,
  ElectricRegulatorMotorized,
  type ElectricRegulatorMotorizedProps,
  type ElectricRegulatorProps,
  Filter,
  type FilterProps,
  FlameArrestor,
  FlameArrestorDetonation,
  type FlameArrestorDetonationProps,
  FlameArrestorExplosion,
  type FlameArrestorExplosionProps,
  FlameArrestorFireRes,
  FlameArrestorFireResDetonation,
  type FlameArrestorFireResDetonationProps,
  type FlameArrestorFireResProps,
  type FlameArrestorProps,
  FlatBladeAgitator,
  type FlatBladeAgitatorProps,
  FlowmeterCoriolis,
  type FlowmeterCoriolisProps,
  FlowmeterElectromagnetic,
  type FlowmeterElectromagneticProps,
  FlowmeterFloatSensor,
  type FlowmeterFloatSensorProps,
  FlowmeterGeneral,
  type FlowmeterGeneralProps,
  FlowmeterNozzle,
  type FlowmeterNozzleProps,
  FlowmeterOrifice,
  type FlowmeterOrificeProps,
  FlowmeterPositiveDisplacement,
  type FlowmeterPositiveDisplacementProps,
  FlowmeterPulse,
  type FlowmeterPulseProps,
  FlowmeterRingPiston,
  type FlowmeterRingPistonProps,
  FlowmeterTurbine,
  type FlowmeterTurbineProps,
  FlowmeterVariableArea,
  type FlowmeterVariableAreaProps,
  FlowmeterVenturi,
  type FlowmeterVenturiProps,
  FlowStraightener,
  type FlowStraightenerProps,
  FourWayValve,
  type FourWayValveProps,
  GateValve,
  type GateValveProps,
  Gauge,
  GaugePreview,
  type GaugeProps,
  HeaterElement,
  type HeaterElementProps,
  HeatExchangerGeneral,
  type HeatExchangerGeneralProps,
  HeatExchangerM,
  type HeatExchangerMProps,
  HeatExchangerStraightTube,
  type HeatExchangerStraightTubeProps,
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
  LiquidRingCompressor,
  type LiquidRingCompressorProps,
  ManualValve,
  type ManualValveProps,
  NeedleValve,
  type NeedleValveProps,
  Nozzle,
  type NozzleProps,
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
  PolygonSymbol,
  type PreviewProps,
  PropellerAgitator,
  type PropellerAgitatorProps,
  Pump,
  type PumpProps,
  Regulator,
  RegulatorManual,
  type RegulatorManualProps,
  type RegulatorProps,
  ReliefValve,
  type ReliefValveProps,
  RemoteActuator,
  type RemoteActuatorProps,
  RollerVaneCompressor,
  type RollerVaneCompressorProps,
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
  Strainer,
  StrainerCone,
  type StrainerConeProps,
  type StrainerProps,
  Switch,
  type SwitchProps,
  type SymbolProps,
  Tank,
  TankPreview,
  type TankProps,
  TextBox,
  TextBoxPreview,
  type TextBoxProps,
  ThreeWayBallValve,
  type ThreeWayBallValveProps,
  ThreeWayValve,
  type ThreeWayValveProps,
  Thruster,
  type ThrusterProps,
  TJunction,
  type TJunctionProps,
  TurboCompressor,
  type TurboCompressorProps,
  VacuumPump,
  type VacuumPumpProps,
  Value,
  ValuePreview,
  type ValueProps,
  Valve,
  type ValveProps,
  Vent,
  type VentProps,
} from "@/schematic/symbol/Symbols";
import { telem } from "@/telem/aether";
import { control } from "@/telem/control/aether";
import { type Theming } from "@/theming";
import { Value as CoreValue } from "@/vis/value";

export interface Spec<P extends object = object> {
  key: Variant;
  name: string;
  Form: FC<SymbolFormProps>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<PreviewProps<P>>;
  zIndex: number;
}

const Z_INDEX_UPPER = 4;
const Z_INDEX_LOWER = 2;

const VARIANTS = [
  "agitator",
  "angledReliefValve",
  "angledSpringLoadedReliefValve",
  "angledValve",
  "ballValve",
  "threeWayBallValve",
  "gateValve",
  "butterflyValveOne",
  "butterflyValveTwo",
  "breatherValve",
  "offPageReference",
  "box",
  "burstDisc",
  "isoBurstDisc",
  "button",
  "cap",
  "cavityPump",
  "checkValve",
  "cylinder",
  "crossBeamAgitator",
  "electricRegulator",
  "electricRegulatorMotorized",
  "filter",
  "flowStraightener",
  "heaterElement",
  "checkValveWithArrow",
  "flatBladeAgitator",
  "flowmeterGeneral",
  "flowmeterElectromagnetic",
  "flowmeterVariableArea",
  "flowmeterCoriolis",
  "flowmeterNozzle",
  "flowmeterVenturi",
  "flowmeterRingPiston",
  "flowmeterPositiveDisplacement",
  "flowmeterTurbine",
  "flowmeterPulse",
  "flowmeterFloatSensor",
  "flowmeterOrifice",
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
  "regulatorManual",
  "reliefValve",
  "rotaryMixer",
  "screwPump",
  "setpoint",
  "solenoidValve",
  "springLoadedReliefValve",
  "staticMixer",
  "switch",
  "tank",
  "polygon",
  "circle",
  "textBox",
  "threeWayValve",
  "vacuumPump",
  "value",
  "gauge",
  "valve",
  "vent",
  "tJunction",
  "crossJunction",
  "heatExchangerGeneral",
  "heatExchangerM",
  "heatExchangerStraightTube",
  "diaphragmPump",
  "ejectionPump",
  "compressor",
  "turboCompressor",
  "rollerVaneCompressor",
  "liquidRingCompressor",
  "ejectorCompressor",
  "centrifugalCompressor",
  "flameArrestor",
  "flameArrestorDetonation",
  "flameArrestorExplosion",
  "flameArrestorFireRes",
  "flameArrestorFireResDetonation",
  "thruster",
  "nozzle",
  "strainer",
  "strainerCone",
  "customActuator",
  "customStatic",
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
  name: "Three Way",
  key: "threeWayValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Three Way"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ThreeWayValve,
  zIndex: Z_INDEX_UPPER,
};

const valve: Spec<ValveProps> = {
  name: "Generic",
  key: "valve",
  Form: CommonToggleForm,
  Symbol: Valve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Valve,
  zIndex: Z_INDEX_UPPER,
};

const solenoidValve: Spec<SolenoidValveProps> = {
  name: "Solenoid",
  key: "solenoidValve",
  Form: CommonToggleForm,
  Symbol: SolenoidValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Solenoid Valve"),
    normallyOpen: false,
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.SolenoidValve,
  zIndex: Z_INDEX_UPPER,
};

const fourWayValve: Spec<FourWayValveProps> = {
  name: "Four Way",
  key: "fourWayValve",
  Form: CommonToggleForm,
  Symbol: FourWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Four Way Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.FourWayValve,
  zIndex: Z_INDEX_UPPER,
};

const angledValve: Spec<AngledValveProps> = {
  name: "Angled",
  key: "angledValve",
  Form: CommonToggleForm,
  Symbol: AngledValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Angled Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.AngledValve,
  zIndex: Z_INDEX_UPPER,
};

const ballValve: Spec<BallValveProps> = {
  name: "Ball",
  key: "ballValve",
  Form: CommonToggleForm,
  Symbol: BallValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Ball Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.BallValve,
  zIndex: Z_INDEX_UPPER,
};

const threeWayBallValve: Spec<ThreeWayBallValveProps> = {
  name: "Three-Way Ball",
  key: "threeWayBallValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayBallValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Three-Way Ball Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ThreeWayBallValve,
  zIndex: Z_INDEX_UPPER,
};

const gateValve: Spec<GateValveProps> = {
  name: "Gate",
  key: "gateValve",
  Form: CommonToggleForm,
  Symbol: GateValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Gate Valve"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.GateValve,
  zIndex: Z_INDEX_UPPER,
};

const butterflyValveOne: Spec<ButterflyValveOneProps> = {
  name: "Butterfly (Remote)",
  key: "butterflyValveOne",
  Form: CommonToggleForm,
  Symbol: ButterflyValveOne,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Butterfly Valve (Remote)"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ButterflyValveOne,
  zIndex: Z_INDEX_UPPER,
};

const butterflyValveTwo: Spec<ButterflyValveTwoProps> = {
  name: "Butterfly (Manual)",
  key: "butterflyValveTwo",
  Form: CommonToggleForm,
  Symbol: ButterflyValveTwo,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Butterfly Valve (Manual)"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.ButterflyValveTwo,
  zIndex: Z_INDEX_UPPER,
};

const breatherValve: Spec<BreatherValveProps> = {
  name: "Breather",
  key: "breatherValve",
  Form: CommonDummyToggleForm,
  Symbol: BreatherValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Breather Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.BreatherValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const pump: Spec<PumpProps> = {
  name: "Pump",
  key: "pump",
  Form: CommonToggleForm,
  Symbol: Pump,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Pump,
  zIndex: Z_INDEX_UPPER,
};

const screwPump: Spec<ScrewPumpProps> = {
  name: "Screw",
  key: "screwPump",
  Form: CommonToggleForm,
  Symbol: ScrewPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
    ...zeroLabel("Tank"),
    borderRadius: DEFAULT_BORDER_RADIUS,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: TankPreview,
  zIndex: Z_INDEX_LOWER,
};

const polygon: Spec<Primitives.PolygonProps> = {
  name: "Polygon",
  key: "polygon",
  Symbol: PolygonSymbol,
  Form: CommonPolygonForm,
  defaultProps: (t) => ({
    numSides: 6,
    sideLength: DEFAULT_POLYGON_SIDE_LENGTH,
    cornerRounding: 0,
    rotation: 0,
    color: t.colors.gray.l11,
    backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
    strokeWidth: 2,
    ...zeroLabel("Polygon"),
  }),
  Preview: removeProps(Primitives.Polygon, ["clickable"]),
  zIndex: Z_INDEX_LOWER,
};

const circle: Spec<Primitives.CircleShapeProps> = {
  name: "Circle",
  key: "circle",
  Symbol: Circle,
  Form: CircleForm,
  defaultProps: (t) => ({
    radius: 20,
    color: t.colors.gray.l11,
    backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
    ...zeroLabel("Circle"),
    strokeWidth: 2,
  }),
  Preview: removeProps(Primitives.CircleShape, ["clickable"]),
  zIndex: Z_INDEX_LOWER,
};

const cylinder: Spec<CylinderProps> = {
  name: "Cylinder",
  key: "cylinder",
  Form: CylinderForm,
  Symbol: Cylinder,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
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
    color: t.colors.gray.l11,
    backgroundColor: color.setAlpha(t.colors.gray.l1, 0),
    ...zeroLabel("Box"),
    borderRadius: ZERO_BOX_BORDER_RADIUS,
    strokeWidth: 2,
    ...ZERO_BOX_PROPS,
    ...ZERO_PROPS,
  }),
  Preview: BoxPreview,
  zIndex: Z_INDEX_LOWER,
};

const reliefValve: Spec<ReliefValveProps> = {
  name: "Relief",
  key: "reliefValve",
  Form: CommonDummyToggleForm,
  Symbol: ReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.ReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const springLoadedReliefValve: Spec<SpringLoadedReliefValveProps> = {
  name: "Spring Loaded Relief",
  key: "springLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: SpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.SpringLoadedReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const angledSpringLoadedReliefValve: Spec<AngledSpringLoadedReliefValveProps> = {
  name: "Angled Spring Loaded Relief",
  key: "angledSpringLoadedReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledSpringLoadedReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Angled Spring Loaded Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.AngledSpringLoadedReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const regulator: Spec<RegulatorProps> = {
  name: "Regulator",
  key: "regulator",
  Form: CommonStyleForm,
  Symbol: Regulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Regulator,
  zIndex: Z_INDEX_UPPER,
};

const regulatorManual: Spec<RegulatorManualProps> = {
  name: "Manual",
  key: "regulatorManual",
  Form: CommonStyleForm,
  Symbol: RegulatorManual,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Manual Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: removeProps(Primitives.RegulatorManual, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const electricRegulator: Spec<ElectricRegulatorProps> = {
  name: "Electric",
  key: "electricRegulator",
  Form: CommonStyleForm,
  Symbol: ElectricRegulator,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Electric Regulator"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ElectricRegulator,
  zIndex: Z_INDEX_UPPER,
};

const electricRegulatorMotorized: Spec<ElectricRegulatorMotorizedProps> = {
  name: "Motorized",
  key: "electricRegulatorMotorized",
  Form: CommonStyleForm,
  Symbol: ElectricRegulatorMotorized,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Electric Regulator Motorized"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ElectricRegulatorMotorized,
  zIndex: Z_INDEX_UPPER,
};

const burstDisc: Spec<BurstDiscProps> = {
  name: "Standard",
  key: "burstDisc",
  Form: CommonStyleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Burst Disc"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.BurstDisc,
  zIndex: Z_INDEX_UPPER,
};

const isoBurstDisc: Spec<ISOBurstDiscProps> = {
  name: "ISO",
  key: "isoBurstDisc",
  Form: CommonStyleForm,
  Symbol: ISOBurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    ...zeroLabel("ISO Cap"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOCap,
  zIndex: Z_INDEX_UPPER,
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual",
  key: "manualValve",
  Form: CommonDummyToggleForm,
  Symbol: ManualValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Manual Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.ManualValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const orificePlate: Spec<OrificePlateProps> = {
  name: "Plate",
  key: "orificePlate",
  Form: CommonStyleForm,
  Symbol: OrificePlate,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    ...zeroLabel("Filter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Filter,
  zIndex: Z_INDEX_UPPER,
};

const flowStraightener: Spec<FlowStraightenerProps> = {
  name: "Flow Straightener",
  key: "flowStraightener",
  Form: CommonStyleForm,
  Symbol: FlowStraightener,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flow Straightener"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowStraightener,
  zIndex: Z_INDEX_UPPER,
};

const heaterElement: Spec<HeaterElementProps> = {
  name: "Heater",
  key: "heaterElement",
  Form: CommonStyleForm,
  Symbol: HeaterElement,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Heater Element"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeaterElement,
  zIndex: Z_INDEX_UPPER,
};

const needleValve: Spec<NeedleValveProps> = {
  name: "Needle",
  key: "needleValve",
  Form: CommonDummyToggleForm,
  Symbol: NeedleValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Needle Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.NeedleValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const checkValve: Spec<CheckValveProps> = {
  name: "Check",
  key: "checkValve",
  Form: CommonStyleForm,
  Symbol: CheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    ...zeroLabel("Orifice"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Orifice,
  zIndex: Z_INDEX_UPPER,
};

const angledReliefValve: Spec<ReliefValveProps> = {
  name: "Angled Relief",
  key: "angledReliefValve",
  Form: CommonDummyToggleForm,
  Symbol: AngledReliefValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Angled Relief Valve"),
    ...ZERO_DUMMY_TOGGLE_PROPS,
  }),
  Preview: removeProps(Primitives.AngledReliefValve, ["clickable"]),
  zIndex: Z_INDEX_UPPER,
};

const value: Spec<ValueProps> = {
  name: "Value",
  key: "value",
  Form: ValueForm,
  Symbol: Value,
  Preview: ValuePreview,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    units: "psi",
    level: "h5",
    inlineSize: 70,
    ...zeroLabel("Value"),
    ...ZERO_PROPS,
    stalenessTimeout: 5,
    stalenessColor: t.colors.warning.m1,
    telem: ZERO_NUMERIC_STRINGER_SOURCE_PROPS.source,
    redline: CoreValue.ZERO_READLINE,
  }),
  zIndex: Z_INDEX_UPPER,
};

const gauge: Spec<GaugeProps> = {
  name: "Gauge",
  key: "gauge",
  Form: GaugeForm,
  Symbol: Gauge,
  Preview: GaugePreview,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    units: "RPM",
    level: "h5",
    bounds: bounds.construct(0, 100),
    barWidth: 10,
    ...zeroLabel("Gauge"),
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
    color: t.colors.primary.z,
    ...zeroLabel("Button"),
    ...ZERO_BOOLEAN_SINK_PROPS,
    mode: "fire",
    onClickDelay: 0,
    scale: null,
  }),
  zIndex: Z_INDEX_UPPER,
};

const switch_: Spec<SwitchProps> = {
  name: "Switch",
  key: "switch",
  Symbol: Switch,
  Form: SwitchForm,
  defaultProps: () => ({
    ...zeroLabel("Switch"),
    ...ZERO_TOGGLE_PROPS,
    scale: null,
  }),
  Preview: Primitives.Switch,
  zIndex: Z_INDEX_UPPER,
};

const vacuumPump: Spec<VacuumPumpProps> = {
  name: "Vacuum",
  key: "vacuumPump",
  Symbol: VacuumPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Vacuum Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.VacuumPump,
  zIndex: Z_INDEX_UPPER,
};

const compressor: Spec<CompressorProps> = {
  name: "Compressor",
  key: "compressor",
  Symbol: Compressor,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Compressor,
  zIndex: Z_INDEX_UPPER,
};

const cavityPump: Spec<CavityPumpProps> = {
  name: "Cavity",
  key: "cavityPump",
  Symbol: CavityPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Cavity Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.CavityPump,
  zIndex: Z_INDEX_UPPER,
};

const pistonPump: Spec<PistonPumpProps> = {
  name: "Piston",
  key: "pistonPump",
  Symbol: PistonPump,
  Form: CommonToggleForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    ...zeroLabel("Helical Agitator"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.HelicalAgitator,
  zIndex: Z_INDEX_UPPER,
};

const textBox: Spec<TextBoxProps> = {
  name: "Text Box",
  key: "textBox",
  Symbol: TextBox,
  Form: TextBoxForm,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    autoFit: true,
    align: "center",
    ...zeroLabel("Text Box"),
    ...ZERO_PROPS,
    ...ZERO_BOX_PROPS,
    level: "p",
    value: "Text Box",
    width: 75,
  }),
  Preview: TextBoxPreview,
  zIndex: Z_INDEX_UPPER,
};

const offPageReference: Spec<OffPageReferenceProps> = {
  name: "Off Page",
  key: "offPageReference",
  Form: OffPageReferenceForm,
  Symbol: OffPageReference,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    orientation: "right",
    ...zeroLabel("Off Page Reference"),
  }),
  Preview: OffPageReferencePreview,
  zIndex: Z_INDEX_UPPER,
};

const isoCheckValve: Spec<ISOCheckValveProps> = {
  name: "ISO Check",
  key: "isoCheckValve",
  Form: CommonStyleForm,
  Symbol: ISOCheckValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("ISO Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.ISOCheckValve,
  zIndex: Z_INDEX_UPPER,
};

const checkValveWithArrow: Spec<CheckValveWithArrowProps> = {
  name: "Check (Arrow)",
  key: "checkValveWithArrow",
  Form: CommonStyleForm,
  Symbol: CheckValveWithArrow,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Check Valve"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.CheckValveWithArrow,
  zIndex: Z_INDEX_UPPER,
};

const vent: Spec<VentProps> = {
  name: "Vent",
  key: "vent",
  Form: CommonStyleForm,
  Symbol: Vent,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
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
    color: t.colors.gray.l11,
    ...zeroLabel(""),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.TJunction,
  zIndex: Z_INDEX_UPPER + 20,
};

const crossJunction: Spec<CrossJunctionProps> = {
  name: "Cross Junction",
  key: "crossJunction",
  Form: CommonStyleForm,
  Symbol: CrossJunction,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel(""),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.CrossJunction,
  zIndex: Z_INDEX_UPPER + 20,
};

const flowmeterGeneral: Spec<FlowmeterGeneralProps> = {
  name: "General",
  key: "flowmeterGeneral",
  Form: CommonStyleForm,
  Symbol: FlowmeterGeneral,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("General Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterGeneral,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterElectromagnetic: Spec<FlowmeterElectromagneticProps> = {
  name: "Electromagnetic",
  key: "flowmeterElectromagnetic",
  Form: CommonStyleForm,
  Symbol: FlowmeterElectromagnetic,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Electromagnetic Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterElectromagnetic,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterVariableArea: Spec<FlowmeterVariableAreaProps> = {
  name: "Variable Area",
  key: "flowmeterVariableArea",
  Form: CommonStyleForm,
  Symbol: FlowmeterVariableArea,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Variable Area Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterVariableArea,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterCoriolis: Spec<FlowmeterCoriolisProps> = {
  name: "Coriolis",
  key: "flowmeterCoriolis",
  Form: CommonStyleForm,
  Symbol: FlowmeterCoriolis,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Coriolis Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterCoriolis,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterNozzle: Spec<FlowmeterNozzleProps> = {
  name: "Nozzle",
  key: "flowmeterNozzle",
  Form: CommonStyleForm,
  Symbol: FlowmeterNozzle,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Nozzle Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterNozzle,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterVenturi: Spec<FlowmeterVenturiProps> = {
  name: "Venturi",
  key: "flowmeterVenturi",
  Form: CommonStyleForm,
  Symbol: FlowmeterVenturi,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Venturi Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterVenturi,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterRingPiston: Spec<FlowmeterRingPistonProps> = {
  name: "Ring Piston",
  key: "flowmeterRingPiston",
  Form: CommonStyleForm,
  Symbol: FlowmeterRingPiston,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Ring Piston Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterRingPiston,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterPositiveDisplacement: Spec<FlowmeterPositiveDisplacementProps> = {
  name: "Positive Displacement",
  key: "flowmeterPositiveDisplacement",
  Form: CommonStyleForm,
  Symbol: FlowmeterPositiveDisplacement,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Positive Displacement Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterPositiveDisplacement,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterTurbine: Spec<FlowmeterTurbineProps> = {
  name: "Turbine",
  key: "flowmeterTurbine",
  Form: CommonStyleForm,
  Symbol: FlowmeterTurbine,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Turbine Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterTurbine,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterPulse: Spec<FlowmeterPulseProps> = {
  name: "Pulse",
  key: "flowmeterPulse",
  Form: CommonStyleForm,
  Symbol: FlowmeterPulse,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Pulse Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterPulse,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterFloatSensor: Spec<FlowmeterFloatSensorProps> = {
  name: "Float Sensor",
  key: "flowmeterFloatSensor",
  Form: CommonStyleForm,
  Symbol: FlowmeterFloatSensor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Float Sensor Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterFloatSensor,
  zIndex: Z_INDEX_UPPER,
};

const flowmeterOrifice: Spec<FlowmeterOrificeProps> = {
  name: "Orifice",
  key: "flowmeterOrifice",
  Form: CommonStyleForm,
  Symbol: FlowmeterOrifice,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Orifice Flowmeter"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlowmeterOrifice,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerGeneral: Spec<HeatExchangerGeneralProps> = {
  name: "Heat Exchanger",
  key: "heatExchangerGeneral",
  Form: CommonStyleForm,
  Symbol: HeatExchangerGeneral,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("General Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerGeneral,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerM: Spec<HeatExchangerMProps> = {
  name: "M-Type Heat Exchanger",
  key: "heatExchangerM",
  Form: CommonStyleForm,
  Symbol: HeatExchangerM,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("M Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerM,
  zIndex: Z_INDEX_UPPER,
};

const heatExchangerStraightTube: Spec<HeatExchangerStraightTubeProps> = {
  name: "Straight Tube Heat Exchanger",
  key: "heatExchangerStraightTube",
  Form: CommonStyleForm,
  Symbol: HeatExchangerStraightTube,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Straight Tube Heat Exchanger"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.HeatExchangerStraightTube,
  zIndex: Z_INDEX_UPPER,
};

const turboCompressor: Spec<TurboCompressorProps> = {
  name: "Turbo Compressor",
  key: "turboCompressor",
  Form: CommonToggleForm,
  Symbol: TurboCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Turbo Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.TurboCompressor,
  zIndex: Z_INDEX_UPPER,
};

const rollerVaneCompressor: Spec<RollerVaneCompressorProps> = {
  name: "Roller Vane Compressor",
  key: "rollerVaneCompressor",
  Form: CommonToggleForm,
  Symbol: RollerVaneCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Roller Vane Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.RollerVaneCompressor,
  zIndex: Z_INDEX_UPPER,
};

const liquidRingCompressor: Spec<LiquidRingCompressorProps> = {
  name: "Liquid Ring Compressor",
  key: "liquidRingCompressor",
  Form: CommonToggleForm,
  Symbol: LiquidRingCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Liquid Ring Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.LiquidRingCompressor,
  zIndex: Z_INDEX_UPPER,
};

const ejectorCompressor: Spec<EjectorCompressorProps> = {
  name: "Ejector Compressor",
  key: "ejectorCompressor",
  Form: CommonToggleForm,
  Symbol: EjectorCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Ejector Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.EjectorCompressor,
  zIndex: Z_INDEX_UPPER,
};

const centrifugalCompressor: Spec<CentrifugalCompressorProps> = {
  name: "Centrifugal Compressor",
  key: "centrifugalCompressor",
  Form: CommonToggleForm,
  Symbol: CentrifugalCompressor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Centrifugal Compressor"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.CentrifugalCompressor,
  zIndex: Z_INDEX_UPPER,
};

const diaphragmPump: Spec<DiaphragmPumpProps> = {
  name: "Diaphragm Pump",
  key: "diaphragmPump",
  Form: CommonToggleForm,
  Symbol: DiaphragmPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Diaphragm Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.DiaphragmPump,
  zIndex: Z_INDEX_UPPER,
};

const ejectionPump: Spec<EjectionPumpProps> = {
  name: "Ejection",
  key: "ejectionPump",
  Form: CommonToggleForm,
  Symbol: EjectionPump,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Ejection Pump"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.EjectionPump,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestor: Spec<FlameArrestorProps> = {
  name: "Standard",
  key: "flameArrestor",
  Form: CommonStyleForm,
  Symbol: FlameArrestor,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flame Arrestor"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestor,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorExplosion: Spec<FlameArrestorExplosionProps> = {
  name: "Explosion-Proof",
  key: "flameArrestorExplosion",
  Form: CommonStyleForm,
  Symbol: FlameArrestorExplosion,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flame Arrestor (Explosion-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorExplosion,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorDetonation: Spec<FlameArrestorDetonationProps> = {
  name: "Detonation-Proof",
  key: "flameArrestorDetonation",
  Form: CommonStyleForm,
  Symbol: FlameArrestorDetonation,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flame Arrestor (Detonation-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorDetonation,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorFireRes: Spec<FlameArrestorFireResProps> = {
  name: "Fire Resistant",
  key: "flameArrestorFireRes",
  Form: CommonStyleForm,
  Symbol: FlameArrestorFireRes,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flame Arrestor (Fire Resistant)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorFireRes,
  zIndex: Z_INDEX_UPPER,
};

const flameArrestorFireResDetonation: Spec<FlameArrestorFireResDetonationProps> = {
  name: "Fire Resistant & Detonation-Proof",
  key: "flameArrestorFireResDetonation",
  Form: CommonStyleForm,
  Symbol: FlameArrestorFireResDetonation,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Flame Arrestor (Fire Resistant and Detonation-Proof)"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.FlameArrestorFireResDetonation,
  zIndex: Z_INDEX_UPPER,
};

const thruster: Spec<ThrusterProps> = {
  name: "Thruster",
  key: "thruster",
  Form: CommonToggleForm,
  Symbol: Thruster,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Thruster"),
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: Primitives.Thruster,
  zIndex: Z_INDEX_UPPER,
};

const nozzle: Spec<NozzleProps> = {
  name: "Nozzle",
  key: "nozzle",
  Form: CommonStyleForm,
  Symbol: Nozzle,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Nozzle"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Nozzle,
  zIndex: Z_INDEX_UPPER,
};

const strainer: Spec<StrainerProps> = {
  name: "Strainer",
  key: "strainer",
  Form: CommonStyleForm,
  Symbol: Strainer,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Strainer"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.Strainer,
  zIndex: Z_INDEX_UPPER,
};

const strainerCone: Spec<StrainerConeProps> = {
  name: "Cone",
  key: "strainerCone",
  Form: CommonStyleForm,
  Symbol: StrainerCone,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Strainer Cone"),
    ...ZERO_PROPS,
  }),
  Preview: Primitives.StrainerCone,
  zIndex: Z_INDEX_UPPER,
};

const customActuator: Spec<RemoteActuatorProps> = {
  name: "Custom Actuator",
  key: "customActuator",
  Form: CommonToggleForm,
  Symbol: RemoteActuator,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Custom Actuator"),
    ...ZERO_TOGGLE_PROPS,
    specKey: "",
    stateOverrides: [],
  }),
  Preview: Primitives.CustomActuator,
  zIndex: Z_INDEX_UPPER,
};

const customStatic: Spec<CustomStaticProps> = {
  name: "Custom Static",
  key: "customStatic",
  Form: CommonStyleForm,
  Symbol: CustomStatic,
  defaultProps: (t) => ({
    color: t.colors.gray.l11,
    ...zeroLabel("Custom Static"),
    ...ZERO_PROPS,
    specKey: "",
    stateOverrides: [],
  }),
  Preview: Primitives.CustomStatic,
  zIndex: Z_INDEX_UPPER,
};

export const REGISTRY: Record<Variant, Spec<any>> = {
  value,
  gauge,
  button,
  tank,
  polygon,
  circle,
  tJunction,
  crossJunction,
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
  ballValve,
  threeWayBallValve,
  gateValve,
  butterflyValveOne,
  butterflyValveTwo,
  breatherValve,
  manualValve,
  needleValve,
  reliefValve,
  angledReliefValve,
  checkValve,
  regulator,
  regulatorManual,
  electricRegulator,
  electricRegulatorMotorized,
  springLoadedReliefValve,
  angledSpringLoadedReliefValve,
  pump,
  pistonPump,
  screwPump,
  cavityPump,
  diaphragmPump,
  ejectionPump,
  vacuumPump,
  compressor,
  turboCompressor,
  rollerVaneCompressor,
  liquidRingCompressor,
  ejectorCompressor,
  centrifugalCompressor,
  staticMixer,
  rotaryMixer,
  burstDisc,
  isoBurstDisc,
  cap,
  isoCap,
  filter,
  isoFilter,
  flowStraightener,
  heaterElement,
  orifice,
  orificePlate,
  agitator,
  propellerAgitator,
  flatBladeAgitator,
  paddleAgitator,
  crossBeamAgitator,
  helicalAgitator,
  isoCheckValve,
  checkValveWithArrow,
  vent,
  cylinder,
  flowmeterGeneral,
  flowmeterElectromagnetic,
  flowmeterVariableArea,
  flowmeterCoriolis,
  flowmeterNozzle,
  flowmeterVenturi,
  flowmeterRingPiston,
  flowmeterPositiveDisplacement,
  flowmeterTurbine,
  flowmeterPulse,
  flowmeterFloatSensor,
  flowmeterOrifice,
  heatExchangerGeneral,
  heatExchangerM,
  heatExchangerStraightTube,
  flameArrestor,
  flameArrestorExplosion,
  flameArrestorDetonation,
  flameArrestorFireRes,
  flameArrestorFireResDetonation,
  thruster,
  nozzle,
  strainer,
  strainerCone,
  customActuator,
  customStatic,
};

export interface Group extends group.Group {
  Icon: Icon.FC;
  symbols: Variant[];
}

export const GROUPS: Group[] = [
  {
    key: "general",
    Icon: Icon.Channel,
    name: "General",
    symbols: [
      "value",
      "gauge",
      "setpoint",
      "textBox",
      "offPageReference",
      "button",
      "switch",
      "light",
      "polygon",
      "circle",
      "box",
    ],
  },
  {
    key: "vessels",
    name: "Vessels",
    Icon: Icon.Tank,
    symbols: ["tank", "cylinder", "tJunction", "crossJunction"],
  },
  {
    key: "valves",
    name: "Valves",
    Icon: Icon.Valve,
    symbols: [
      "valve",
      "solenoidValve",
      "threeWayValve",
      "fourWayValve",
      "angledValve",
      "ballValve",
      "threeWayBallValve",
      "gateValve",
      "butterflyValveOne",
      "butterflyValveTwo",
      "breatherValve",
      "manualValve",
      "needleValve",
      "reliefValve",
      "angledReliefValve",
      "springLoadedReliefValve",
      "angledSpringLoadedReliefValve",
      "checkValve",
      "isoCheckValve",
      "checkValveWithArrow",
      "regulator",
      "regulatorManual",
      "electricRegulator",
      "electricRegulatorMotorized",
    ],
  },
  {
    key: "pumps",
    name: "Pumps",
    Icon: Icon.Pump,
    symbols: [
      "pump",
      "screwPump",
      "pistonPump",
      "cavityPump",
      "diaphragmPump",
      "ejectionPump",
      "vacuumPump",
      "compressor",
      "turboCompressor",
      "rollerVaneCompressor",
      "liquidRingCompressor",
      "ejectorCompressor",
      "centrifugalCompressor",
    ],
  },
  {
    key: "meters",
    name: "Flow Meters",
    Icon: Icon.Rule,
    symbols: [
      "flowmeterGeneral",
      "flowmeterElectromagnetic",
      "flowmeterVariableArea",
      "flowmeterCoriolis",
      "flowmeterNozzle",
      "flowmeterVenturi",
      "flowmeterRingPiston",
      "flowmeterPositiveDisplacement",
      "flowmeterTurbine",
      "flowmeterPulse",
      "flowmeterFloatSensor",
      "flowmeterOrifice",
    ],
  },
  {
    key: "process",
    name: "Process",
    Icon: Icon.Process,
    symbols: [
      "heatExchangerGeneral",
      "heatExchangerM",
      "heatExchangerStraightTube",
      "staticMixer",
      "rotaryMixer",
      "agitator",
      "propellerAgitator",
      "flatBladeAgitator",
      "paddleAgitator",
      "crossBeamAgitator",
      "helicalAgitator",
    ],
  },
  {
    key: "safety",
    name: "Safety",
    Icon: Icon.Safety,
    symbols: [
      "burstDisc",
      "isoBurstDisc",
      "flameArrestor",
      "flameArrestorDetonation",
      "flameArrestorExplosion",
      "flameArrestorFireRes",
      "flameArrestorFireResDetonation",
    ],
  },
  {
    key: "fittings",
    name: "Fittings",
    Icon: Icon.Fitting,
    symbols: [
      "cap",
      "isoCap",
      "orifice",
      "orificePlate",
      "vent",
      "nozzle",
      "heaterElement",
      "thruster",
      "filter",
      "isoFilter",
      "strainer",
      "strainerCone",
      "flowStraightener",
    ],
  },
];
