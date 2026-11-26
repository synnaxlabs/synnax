// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, schematic } from "@synnaxlabs/client";
import { deep, record } from "@synnaxlabs/x";
import type { z } from "zod";

import { NAMES } from "@/schematic/node/builtIns";
import {
  CUSTOM_ACTUATOR_VARIANT,
  CUSTOM_STATIC_VARIANT,
  customActuatorSpec,
  customStaticSpec,
} from "@/schematic/node/custom/configs";
import { General } from "@/schematic/node/general";
import { GroupBox } from "@/schematic/node/groupBox";
import type { Spec } from "@/schematic/node/spec";
import { SYMBOL_VARIANT, symbolSpec } from "@/schematic/node/symbol/spec";
import { Vessels } from "@/schematic/node/vessels";

export const REGISTRY = {
  ...General.REGISTRY,
  ...Vessels.REGISTRY,
  custom_actuator: customActuatorSpec,
  custom_static: customStaticSpec,
  group_box: GroupBox.spec,
  symbol: symbolSpec,
} as const satisfies Partial<Record<schematic.NodeConfigType, unknown>>;

export const variantZ = schematic.nodeConfigTypeZ;
export type Variant = schematic.NodeConfigType;

export const configZ = schematic.nodeConfigZ;
export type Config = schematic.NodeConfig;
export type ConfigOf<V extends Variant> = Extract<Config, { variant: V }>;

/**
 * Input is the schema input for a variant: its discriminator plus its fields, of which
 * those with a schema default are optional and the rest required.
 */
export type Input<V extends Variant> = { variant: V } & Omit<
  Extract<z.input<typeof configZ>, { variant: V }>,
  "variant"
>;

export const resolveSpec = (variant: string): Spec => {
  const spec = REGISTRY[variant as keyof typeof REGISTRY];
  if (spec == null) throw new NotFoundError(`Symbol with variant ${variant} not found`);
  return spec as Spec;
};

/** Whether the config belongs to a node variant rather than an edge. */
export const isConfig = (config: schematic.ElementConfig): config is Config =>
  config.variant in REGISTRY;

/**
 * Builds a fresh config from the schema input. Every unset value comes from the
 * schema, except the label, which names the symbol unless the input sets it, and the
 * value field, which takes the spec's value unless the input sets it.
 * @param input - The variant plus any fields to set on top of the schema defaults.
 * @throws {NotFoundError} if no spec is registered for the variant.
 */
export const createConfig = <V extends Variant>(input: Input<V>): ConfigOf<V> => {
  const config = configZ.parse(input) as ConfigOf<V>;
  const spec = resolveSpec(input.variant);
  const labeled = input as { label?: { label?: string } };
  if ("label" in config && labeled.label?.label == null)
    config.label.label = spec.label ?? spec.name;
  if ("value" in config && spec.value != null && !("value" in input))
    config.value = spec.value;
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

/** @returns the variant that draws the custom symbol. */
export const customVariant = (symbol: schematic.symbol.Symbol): CustomVariant =>
  symbol.data.variant === "actuator" ? CUSTOM_ACTUATOR_VARIANT : CUSTOM_STATIC_VARIANT;

/**
 * @returns the key of the symbol the config draws: the key of a custom symbol, or the
 * name of a built-in one.
 */
export const symbolKey = (config: Config): string => {
  if (isCustomConfig(config)) return config.specKey;
  if (config.variant !== SYMBOL_VARIANT) return config.variant;
  return config.spec.type === "builtin" ? config.spec.name : config.spec.key;
};

/**
 * @returns the label that a node of the built-in symbol takes on creation, or undefined
 * if no built-in has the name.
 */
export const builtInLabel = (name: string): string | undefined =>
  NAMES.get(name)?.label;

/**
 * @returns the name that the symbols toolbar gives the built-in symbol, or undefined if
 * no built-in has the name.
 */
export const builtInName = (name: string): string | undefined => NAMES.get(name)?.name;

/**
 * Changes the config to the variant the input names. Each field the new variant
 * accepts keeps its value, except the symbol that the config draws and the state
 * overrides of a custom symbol.
 * @param config - The config to change.
 * @param input - The variant to change to, plus any fields to set on top.
 * @throws {NotFoundError} if no spec is registered for the variant.
 */
export const changeConfig = <V extends Variant>(
  config: Config,
  input: Input<V>,
): ConfigOf<V> =>
  deep.overrideValidItems(
    createConfig(input),
    record.omit(config as record.Unknown, "spec", "specKey", "stateOverrides"),
    schematic.NODE_CONFIG_SCHEMAS[input.variant],
  );
