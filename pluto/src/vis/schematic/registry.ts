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
  LightForm,
  SolenoidValveForm,
  type SymbolFormProps,
  TankForm,
  ValueForm,
} from "@/vis/schematic/Forms";
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
  Filter,
  FilterPreview,
  type FilterProps,
  FourWayValve,
  FourWayValvePreview,
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
  "filter",
  "fourWayValve",
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

const typeZ = z.enum(VARIANTS);
export type Variant = z.infer<typeof typeZ>;

const ZERO_TOGGLE_PROPS = {
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

const threeWayValve: Spec<ThreeWayValveProps> = {
  name: "Three Way Valve",
  key: "threeWayValve",
  Form: CommonToggleForm,
  Symbol: ThreeWayValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    label: {
      label: "Three Way Valve",
      level: "p",
      orientation: "top",
    },
    ...ZERO_TOGGLE_PROPS,
    orientation: "left",
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
    label: {
      label: "Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Solenoid Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Four Way Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Angled Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Pump",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Screw Pump",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Tank",
      level: "p",
      orientation: "top",
    },
    dimensions: {
      width: 100,
      height: 200,
    },
    orientation: "left",
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
    label: {
      label: "Relief Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Regulator",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
  }),
  Preview: RegulatorPreview,
  zIndex: Z_INDEX_UPPER,
};

const burstDisc: Spec<ReliefValveProps> = {
  name: "Burst Disc",
  key: "burstDisc",
  Form: CommonNonToggleForm,
  Symbol: BurstDisc,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    label: {
      label: "Burst Disc",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Cap",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Manual Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Filter",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Needle Valve",
      level: "p",
    },
    orientation: "left",
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
    label: {
      label: "Check Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Orifice",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Angled Relief Valve",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Value",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
    telem: telem.sourcePipeline("string", {
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
        stringifier: telem.stringifyNumber({ precision: 2 }),
        rollingAverage: telem.rollingAverage({ windowSize: 1 }),
      },
      outlet: "stringifier",
    }),
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
    label: {
      label: "Button",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Switch",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Vacuum Pump",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Cavity Pump",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Piston Pump",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Static Mixer",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Rotary Mixer",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
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
    label: {
      label: "Light",
      level: "p",
      orientation: "top",
    },
    orientation: "left",
    units: "psi",
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
  }),
  Preview: LightPreview,
  zIndex: Z_INDEX_UPPER,
};

export const SYMBOLS: Record<Variant, Spec<any>> = {
  value,
  switch: switch_,
  button,
  tank,
  valve,
  threeWayValve,
  solenoidValve,
  fourWayValve,
  angledValve,
  light,
  manualValve,
  needleValve,
  reliefValve,
  checkValve,
  regulator,
  angledReliefValve,
  pistonPump,
  pump,
  screwPump,
  vacuumPump,
  cavityPump,
  staticMixer,
  rotaryMixer,
  burstDisc,
  cap,
  filter,
  orifice,
};
