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
} from "@/vis/pid/Forms";
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
} from "@/vis/pid/Symbols";

export interface Spec<P extends object> {
  name: string;
  variant: string;
  Form: FC<SymbolFormProps<P>>;
  Symbol: FC<SymbolProps<P>>;
  defaultProps: (t: Theming.Theme) => P;
  Preview: FC<SymbolProps<P>>;
}

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
  variant: "threeWayValve",
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
};

const valve: Spec<ValveProps> = {
  name: "Valve",
  variant: "valve",
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
};

const solenoidValve: Spec<SolenoidValveProps> = {
  name: "Solenoid Valve",
  variant: "solenoidValve",
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
    ...ZERO_TOGGLE_PROPS,
  }),
  Preview: SolenoidValvePreview,
};

const fourWayValve: Spec<ValveProps> = {
  name: "Four Way Valve",
  variant: "fourWayValve",
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
  }),
  Preview: FourWayValvePreview,
};

const angledValve: Spec<AngledValveProps> = {
  name: "Angled Valve",
  variant: "angledValve",
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
  }),
  Preview: AngledValvePreview,
};

const pump: Spec<PumpProps> = {
  name: "Pump",
  variant: "pump",
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
  }),
  Preview: PumpPreview,
};

const tank: Spec<TankProps> = {
  name: "Tank",
  variant: "tank",
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
};

const reliefValve: Spec<ReliefValveProps> = {
  name: "Relief Valve",
  variant: "reliefValve",
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
};

const regulator: Spec<RegulatorProps> = {
  name: "Regulator",
  variant: "regulator",
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
};

const burstDisc: Spec<ReliefValveProps> = {
  name: "Burst Disc",
  variant: "burstDisc",
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
};

const cap: Spec<ReliefValveProps> = {
  name: "Cap",
  variant: "cap",
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
};

const manualValve: Spec<ManualValveProps> = {
  name: "Manual Valve",
  variant: "manualValve",
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
};

const filter: Spec<FilterProps> = {
  name: "Filter",
  variant: "filter",
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
};

const needleValve: Spec<NeedleValveProps> = {
  name: "Needle Valve",
  variant: "needleValve",
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
};

const checkValve: Spec<CheckValveProps> = {
  name: "Check Valve",
  variant: "checkValve",
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
};

const orifice: Spec<OrificeProps> = {
  name: "Orifice",
  variant: "orifice",
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
};

const angledReliefValve: Spec<ReliefValveProps> = {
  name: "Angled Relief Valve",
  variant: "angledReliefValve",
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
};

const value: Spec<ValueProps> = {
  name: "Value",
  variant: "value",
  Form: ValueForm,
  Symbol: Value,
  Preview: ValuePreview,
  defaultProps: (t) => ({
    color: t.colors.gray.l9.rgba255,
    level: "small",
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
          to: "stringify",
        },
      ],
      segments: {
        valueStream: telem.streamChannelValue({ channel: 0 }),
        stringify: telem.stringifyNumber({ precision: 2 }),
      },
      outlet: "stringify",
    }),
  }),
};

const button: Spec<ButtonProps> = {
  name: "Button",
  variant: "button",
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
};

const switch_: Spec<SwitchProps> = {
  name: "Switch",
  variant: "switch",
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
