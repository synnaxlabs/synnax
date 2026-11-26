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

import { BUILT_INS } from "@/schematic/node/builtIns";
import { Fittings } from "@/schematic/node/fittings";
import { Flowmeters } from "@/schematic/node/flowmeters";
import { General } from "@/schematic/node/general";
import { GroupBox } from "@/schematic/node/groupBox";
import { Process } from "@/schematic/node/process";
import { Pumps } from "@/schematic/node/pumps";
import { Safety } from "@/schematic/node/safety";
import type { Spec } from "@/schematic/node/spec";
import { SYMBOL_VARIANT, symbolSpec } from "@/schematic/node/symbol/spec";
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
  const spec: unknown = REGISTRY[variant as keyof typeof REGISTRY];
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

/**
 * @returns the key of the symbol the config draws: the key of a custom symbol, or the
 * name of a built-in one.
 */
export const symbolKey = (config: Config): string => {
  if (config.variant !== SYMBOL_VARIANT) return config.variant;
  return config.spec.type === "builtin" ? config.spec.name : config.spec.key;
};

// The labels of the built-ins that no older variant draws.
const LABELS: Record<string, string> = {
  normally_open_solenoid_valve: "Normally open solenoid",
};

/**
 * @returns the label that a node of the built-in symbol takes on creation, or undefined
 * if no built-in has the name.
 */
export const builtInLabel = (name: string): string | undefined => {
  if (!BUILT_INS.has(name)) return undefined;
  if (name in LABELS) return LABELS[name];
  const spec = resolveSpec(name);
  return spec.label ?? spec.name;
};

/**
 * @returns the name that the symbols toolbar gives the built-in symbol, or undefined if
 * no built-in has the name.
 */
export const builtInName = (name: string): string | undefined => {
  if (!BUILT_INS.has(name)) return undefined;
  if (name in LABELS) return LABELS[name];
  return resolveSpec(name).name;
};

/**
 * Changes the config to the variant the input names. Each field the new variant
 * accepts keeps its value, except the symbol that the config draws.
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
    record.omit(config as record.Unknown, "spec"),
    schematic.NODE_CONFIG_SCHEMAS[input.variant],
  );

/// STATIC_SPECS lists every Spec in the registry that is NOT a custom-symbol variant,
/// the group box, or the symbol variant.
export const STATIC_SPECS: readonly Spec[] = (Object.values(REGISTRY) as Spec[]).filter(
  (s) => s.key !== GroupBox.VARIANT && s.key !== SYMBOL_VARIANT,
);
