// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel, schematic } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";

import { Telem } from "@/schematic/node/common/telem";
import type { telem } from "@/telem/aether";
import type { symbolNode } from "@/vis/symbolNode/aether";

/**
 * @returns base with the enumeration, match, bounds, and default of each state replaced
 * by those that its binding sets.
 */
export const spec = (
  base: schematic.symbol.Spec,
  bindings: Omit<schematic.StateBinding, keyof schematic.FormatConfig | "source">[],
): schematic.symbol.Spec => {
  const byKey = new Map(bindings.map((b) => [b.key, b]));
  return {
    ...base,
    states: base.states.map((state) => {
      const b = byKey.get(state.key);
      if (b == null) return state;
      if (state.type === "number")
        return { ...state, bounds: b.bounds ?? state.bounds };
      return {
        ...state,
        enumeration: b.enumeration ?? state.enumeration,
        match: b.match ?? state.match,
        default: b.default ?? state.default,
      };
    }),
  };
};

type Sourced = Pick<schematic.StateBinding, "key" | "source">;

/** @returns the channels that the bound states read, and their command channels. */
export const stateChannels = (bindings: Sourced[]): channel.Key[] =>
  unique.unique(
    bindings.flatMap(({ source }) =>
      source?.type === "channel"
        ? [source.key, source.command].filter((ch) => ch != null)
        : [],
    ),
  );

/** @returns a number source for each channel in channels, by its key. */
export const sources = (
  channels: channel.Key[],
): Record<string, telem.NumberSourceSpec> =>
  Object.fromEntries(channels.map((ch) => [ch, Telem.numberSource(ch)]));

// Readings are keyed by channel.
const reading = (
  readings: Record<string, symbolNode.Reading>,
  source: schematic.StateBinding["source"],
): symbolNode.Reading | undefined =>
  source?.type === "channel" &&
  source.key != null &&
  Object.hasOwn(readings, source.key)
    ? readings[source.key]
    : undefined;

/**
 * @returns the value of each bound state that has one: the reading of its channel, or
 * its local value. A state with no value draws its default.
 */
export const values = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): Map<string, number> => {
  const out = new Map<string, number>();
  for (const { key, source } of bindings) {
    const v =
      source?.type === "local" ? source.value : reading(readings, source)?.value;
    if (v != null) out.set(key, v);
  }
  return out;
};

/** @returns whether a bound state reads a stale channel. */
export const stale = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): boolean => bindings.some(({ source }) => reading(readings, source)?.stale === true);

/**
 * @returns whether a state reads a channel that has no data yet, or whose newest value
 * is NaN.
 */
export const dataMissing = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): boolean =>
  bindings.some(
    ({ source }) =>
      source?.type === "channel" &&
      source.key != null &&
      reading(readings, source)?.value == null,
  );

/** @returns the value of the command channel of each bound state that has one. */
export const commanded = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): Map<string, number> => {
  const out = new Map<string, number>();
  for (const { key, source } of bindings) {
    if (source?.type !== "channel" || source.command == null) continue;
    const v = Object.hasOwn(readings, source.command)
      ? readings[source.command].value
      : null;
    if (v != null) out.set(key, v);
  }
  return out;
};
