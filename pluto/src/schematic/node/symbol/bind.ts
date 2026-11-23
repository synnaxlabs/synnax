// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";

import { Telem } from "@/schematic/node/common/telem";
import type { telem } from "@/telem/aether";
import type { symbolNode } from "@/vis/symbolNode/aether";

/**
 * @returns spec with the enumeration, match, bounds, and default of each state replaced
 * by those that its binding sets.
 */
export const bindSpec = (
  spec: schematic.symbol.Spec,
  bindings: Omit<schematic.StateBinding, keyof schematic.FormatConfig | "source">[],
): schematic.symbol.Spec => {
  const byKey = new Map(bindings.map((b) => [b.key, b]));
  return {
    ...spec,
    states: spec.states.map((state) => {
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

/** @returns a number source for each binding that reads a channel, by its state key. */
export const sources = (
  bindings: Pick<schematic.StateBinding, "key" | "source">[],
): Record<string, telem.NumberSourceSpec> =>
  Object.fromEntries(
    bindings.flatMap(({ key, source }) =>
      source?.type === "channel" && source.key != null
        ? [[key, Telem.numberSource(source.key)]]
        : [],
    ),
  );

/**
 * @returns the value of each bound state that has one: the reading of its channel, or
 * its local value. A state with no value draws its default.
 */
export const values = (
  bindings: Pick<schematic.StateBinding, "key" | "source">[],
  readings: Record<string, symbolNode.Reading>,
): Map<string, number> => {
  const out = new Map<string, number>();
  for (const { key, source } of bindings)
    if (source?.type === "local") out.set(key, source.value);
    else if (source?.type === "channel" && Object.hasOwn(readings, key)) {
      const { value } = readings[key];
      if (value != null) out.set(key, value);
    }
  return out;
};

/**
 * @returns whether a state reads a channel that has no data yet, or whose newest value
 * is NaN.
 */
export const dataMissing = (
  bindings: Pick<schematic.StateBinding, "key" | "source">[],
  readings: Record<string, symbolNode.Reading>,
): boolean =>
  bindings.some(
    ({ key, source }) =>
      source?.type === "channel" &&
      source.key != null &&
      (!Object.hasOwn(readings, key) || readings[key].value == null),
  );
