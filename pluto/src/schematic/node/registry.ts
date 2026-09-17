// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";
import { type z } from "zod";

import {
  CUSTOM_ACTUATOR_VARIANT,
  CUSTOM_STATIC_VARIANT,
  customActuatorSpec,
  customStaticSpec,
} from "@/schematic/node/custom/configs";
import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { GroupBox } from "@/schematic/node/groupBox";
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
  custom_actuator: customActuatorSpec,
  custom_static: customStaticSpec,
  group_box: GroupBox.spec,
} as const satisfies Record<schematic.NodeConfigType, unknown>;

export const variantZ = schematic.nodeConfigTypeZ;
export type Variant = schematic.NodeConfigType;

export const configZ = schematic.nodeConfigZ;
export type Config = schematic.NodeConfig;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

/**
 * Overrides is the schema input for a variant without its discriminator. Fields with
 * a schema default are optional; the rest are required.
 */
export type Overrides<V extends Variant> = Omit<
  Extract<z.input<typeof configZ>, { variant: V }>,
  "variant"
>;

/**
 * CreateArgs is the trailing argument list of {@link createConfig}: the overrides are
 * optional only when the variant has no required field.
 */
export type CreateArgs<V extends Variant> =
  {} extends Overrides<V> ? [overrides?: Overrides<V>] : [overrides: Overrides<V>];

export const resolveSpec = (variant: string): Spec<Variant, Config> => {
  const spec = REGISTRY[variant as Variant];
  if (spec == null) throw new NotFoundError(`Symbol with variant ${variant} not found`);
  return spec as Spec<Variant, Config>;
};

/**
 * Builds a fresh config for the variant. Every value comes from the schema, except the
 * label, which names the symbol unless the overrides set it.
 * @param variant - The node variant.
 * @param overrides - Fields to set on top of the schema defaults. Required when the
 * variant has a field with no default, such as the custom symbols' specKey.
 * @throws {NotFoundError} if no spec is registered for the variant.
 */
export const createConfig = <V extends Variant>(
  variant: V,
  ...[overrides]: CreateArgs<V>
): ConfigOf<V> => {
  const config = configZ.parse({ variant, ...overrides }) as ConfigOf<V>;
  const spec = resolveSpec(variant);
  const labeled = overrides as { label?: { label?: string } } | undefined;
  if ("label" in config && labeled?.label?.label == null)
    config.label.label = spec.label ?? spec.name;
  return config;
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
// groupBox is excluded by key: it is created only by grouping, and a spec-level
// hidden flag is not worth the plumbing for one symbol.
export const STATIC_SPECS: readonly Spec[] = (
  Object.values(REGISTRY) as ReadonlyArray<Spec>
).filter((s) => !isCustomVariant(s.key) && s.key !== GroupBox.VARIANT);
