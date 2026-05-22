// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import z from "zod";

import { Fittings } from "@/schematic/node/fittings";
import { Box } from "@/schematic/node/general/box";
import { Button } from "@/schematic/node/general/button";
import { Circle } from "@/schematic/node/general/circle";
import { CustomActuator } from "@/schematic/node/general/customActuator";
import { CustomStatic } from "@/schematic/node/general/customStatic";
import { Gauge } from "@/schematic/node/general/gauge";
import { Input } from "@/schematic/node/general/input";
import { Light } from "@/schematic/node/general/light";
import { OffPageReference } from "@/schematic/node/general/offPageReference";
import { Polygon } from "@/schematic/node/general/polygon";
import { Select } from "@/schematic/node/general/select";
import { Setpoint } from "@/schematic/node/general/setpoint";
import { StateIndicator } from "@/schematic/node/general/stateIndicator";
import { Switch } from "@/schematic/node/general/switch";
import { TextBox } from "@/schematic/node/general/textBox";
import { Value } from "@/schematic/node/general/value";
import { Meters } from "@/schematic/node/meters";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { Safety } from "@/schematic/node/safety";
import { type Spec } from "@/schematic/node/spec";
import { Valves } from "@/schematic/node/valves";
import { CrossJunction } from "@/schematic/node/vessels/crossJunction";
import { Cylinder } from "@/schematic/node/vessels/cylinder";
import { Tank } from "@/schematic/node/vessels/tank";
import { TJunction } from "@/schematic/node/vessels/tJunction";

export const REGISTRY = {
  ...Fittings.REGISTRY,
  ...Meters.REGISTRY,
  ...Process.REGISTRY,
  ...Pumps.REGISTRY,
  ...Safety.REGISTRY,
  ...Valves.REGISTRY,
  box: Box.spec,
  button: Button.spec,
  circle: Circle.spec,
  customActuator: CustomActuator.spec,
  customStatic: CustomStatic.spec,
  gauge: Gauge.spec,
  input: Input.spec,
  light: Light.spec,
  offPageReference: OffPageReference.spec,
  polygon: Polygon.spec,
  select: Select.spec,
  setpoint: Setpoint.spec,
  stateIndicator: StateIndicator.spec,
  switch: Switch.spec,
  textBox: TextBox.spec,
  value: Value.spec,
  crossJunction: CrossJunction.spec,
  cylinder: Cylinder.spec,
  tank: Tank.spec,
  tJunction: TJunction.spec,
} as const;

const VARIANTS = Object.keys(REGISTRY);
export const variantZ = z.enum(VARIANTS as [string, ...string[]]);
export type Variant = keyof typeof REGISTRY;

export const configZ = z.discriminatedUnion("variant", [
  ...Fittings.configZ.options,
  ...Meters.configZ.options,
  ...Process.configZ.options,
  ...Pumps.configZ.options,
  ...Safety.configZ.options,
  ...Valves.configZ.options,
  Box.configZ,
  Button.configZ,
  Circle.configZ,
  CustomActuator.configZ,
  CustomStatic.configZ,
  Gauge.configZ,
  Input.configZ,
  Light.configZ,
  OffPageReference.configZ,
  Polygon.configZ,
  Select.configZ,
  Setpoint.configZ,
  StateIndicator.configZ,
  Switch.configZ,
  TextBox.configZ,
  Value.configZ,
  CrossJunction.configZ,
  Cylinder.configZ,
  Tank.configZ,
  TJunction.configZ,
]);
export type Config = z.infer<typeof configZ>;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

export const resolveSpec = (variant: string): Spec<Variant, Config> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Symbol with variant ${variant} not found`);
  return spec as unknown as Spec<Variant, Config>;
};

/// CustomVariant is the union of Variants that reference a user-defined
/// symbol spec via specKey rather than rendering a hard-coded SVG.
export type CustomVariant = typeof CustomActuator.VARIANT | typeof CustomStatic.VARIANT;
export type CustomConfig = ConfigOf<CustomVariant>;

/// CUSTOM_VARIANTS is the set form of CustomVariant. Prefer the isCustomVariant
/// / isCustomConfig guards at call sites; expose the set for cases that need
/// to iterate the membership directly (e.g. tests).
export const CUSTOM_VARIANTS: ReadonlySet<Variant> = new Set<Variant>([
  CustomActuator.VARIANT,
  CustomStatic.VARIANT,
]);

export const isCustomVariant = (
  variant: string | undefined,
): variant is CustomVariant =>
  variant != null && CUSTOM_VARIANTS.has(variant as Variant);

export const isCustomConfig = (config: Config): config is CustomConfig =>
  isCustomVariant(config.variant);

/// STATIC_SPECS lists every Spec in the registry that is NOT a custom-symbol
/// variant. Used by the symbols toolbar to render the built-in catalog.
export const STATIC_SPECS: readonly Spec[] = (
  Object.values(REGISTRY) as ReadonlyArray<Spec>
).filter((s) => !isCustomVariant(s.key));
