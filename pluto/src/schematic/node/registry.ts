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

import {
  CUSTOM_ACTUATOR_VARIANT,
  CUSTOM_STATIC_VARIANT,
  customActuatorConfigZ,
  customActuatorSpec,
  customStaticConfigZ,
  customStaticSpec,
} from "@/schematic/node/custom/configs";
import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { Safety } from "@/schematic/node/safety";
import { type Spec } from "@/schematic/node/spec";
import { Valves } from "@/schematic/node/valves";
import { Vessels } from "@/schematic/node/vessels";

export const REGISTRY = {
  ...Fittings.REGISTRY,
  ...Flowmeters.REGISTRY,
  ...General.REGISTRY,
  ...Process.REGISTRY,
  ...Pumps.REGISTRY,
  ...Safety.REGISTRY,
  ...Valves.REGISTRY,
  ...Vessels.REGISTRY,
  customActuator: customActuatorSpec,
  customStatic: customStaticSpec,
} as const;

const VARIANTS = Object.keys(REGISTRY);
export const variantZ = z.enum(VARIANTS);
export type Variant = keyof typeof REGISTRY;

export const configZ = z.discriminatedUnion("variant", [
  ...Fittings.configZ.options,
  ...Flowmeters.configZ.options,
  ...General.configZ.options,
  ...Process.configZ.options,
  ...Pumps.configZ.options,
  ...Safety.configZ.options,
  ...Valves.configZ.options,
  ...Vessels.configZ.options,
  customActuatorConfigZ,
  customStaticConfigZ,
]);
export type Config = z.infer<typeof configZ>;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

export const resolveSpec = (variant: string): Spec<Variant, Config> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Symbol with variant ${variant} not found`);
  return spec as Spec<Variant, Config>;
};

/// CustomVariant is the union of Variants that reference a user-defined
/// symbol spec via specKey rather than rendering a hard-coded SVG.
export type CustomVariant =
  typeof CUSTOM_ACTUATOR_VARIANT | typeof CUSTOM_STATIC_VARIANT;
export type CustomConfig = ConfigOf<CustomVariant>;

/// CUSTOM_VARIANTS is the set form of CustomVariant. Prefer the isCustomVariant
/// / isCustomConfig guards at call sites; expose the set for cases that need
/// to iterate the membership directly (e.g. tests).
export const CUSTOM_VARIANTS: ReadonlySet<Variant> = new Set<Variant>([
  CUSTOM_ACTUATOR_VARIANT,
  CUSTOM_STATIC_VARIANT,
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
