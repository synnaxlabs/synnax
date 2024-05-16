// Copyright 2023 Synnax Labs, Inc.
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
  CommonToggleForm,
  type SymbolFormProps,
  SolenoidValveForm,
  TankForm,
  CommonNonToggleForm,
  ValueForm,
  ButtonForm,
} from "@/vis/schematic/Forms";
import {
  type ThreeWayValveProps,
  type SymbolProps,
  ThreeWayValve,
  ThreeWayValvePreview,
  type ValveProps,
  Valve,
  ValvePreview,
  type SolenoidValveProps,
  SolenoidValve,
  SolenoidValvePreview,
  FourWayValve,
  FourWayValvePreview,
  type AngledValveProps,
  type PumpProps,
  AngledValve,
  Pump,
  Tank,
  ReliefValve,
  ReliefValvePreview,
  TankPreview,
  type ReliefValveProps,
  type TankProps,
  type RegulatorProps,
  RegulatorPreview,
  Regulator,
  BurstDiscPreview,
  Cap,
  CapPreview,
  ManualValvePreview,
  type FilterProps,
  Filter,
  FilterPreview,
  type NeedleValveProps,
  NeedleValvePreview,
  type CheckValveProps,
  CheckValve,
  CheckValvePreview,
  BurstDisc,
  type ManualValveProps,
  ManualValve,
  NeedleValve,
  type OrificeProps,
  Orifice,
  OrificePreview,
  PumpPreview,
  AngledReliefValve,
  AngledValvePreview,
  AngledReliefValvePreview,
  type ValueProps,
  Value,
  ValuePreview,
  type ButtonProps,
  Button,
  ButtonPreview,
  Switch,
  SwitchPreview,
  type SwitchProps,
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
  "threeWayValve",
  "valve",
  "solenoidValve",
  "fourWayValve",
  "angledValve",
  "pump",
  "tank",
  "reliefValve",
  "regulator",
  "burstDisc",
  "cap",
  "manualValve",
  "filter",
  "needleValve",
  "checkValve",
  "orifice",
  "angledReliefValve",
  "value",
  "button",
  "switch",
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
  name: "Pneumatic Valve",
  key: "solenoidValve",
  Form: SolenoidValveForm,
  Symbol: SolenoidValve,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    label: {
      label: "Pneumatic Valve",
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
  defaultProps: (t) => ({
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
  defaultProps: (t) => ({
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

export const SYMBOLS: Record<Variant, Spec<any>> = {
  value,
  threeWayValve,
  valve,
  solenoidValve,
  fourWayValve,
  angledValve,
  pump,
  tank,
  reliefValve,
  regulator,
  burstDisc,
  cap,
  manualValve,
  filter,
  needleValve,
  checkValve,
  orifice,
  angledReliefValve,
  button,
  switch: switch_,
};
