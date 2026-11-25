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

import type { Custom } from "@/schematic/node/common/custom";
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

/** @returns each channel that a bound state reads or commands, once. */
export const channels = (bindings: Sourced[]): channel.Key[] =>
  unique.unique(
    bindings.flatMap(({ source }) =>
      source?.type === "channel"
        ? [source.key, source.command].filter((key) => key != null)
        : [],
    ),
  );

/** Target is where a command writes: a channel, or the local value of a state. */
export type Target =
  { type: "channel"; channel: channel.Key } | { type: "local"; state: string };

/**
 * @returns where command writes: the command channel of its state, the local value of
 * its state, or the channel that commands binds it to. Undefined when it has none.
 */
export const target = (
  command: Pick<schematic.symbol.Command, "key" | "state">,
  states: Sourced[],
  commands: schematic.CommandBinding[],
): Target | undefined => {
  if (command.state == null) {
    const binding = commands.find(({ key }) => key === command.key);
    return binding == null ? undefined : { type: "channel", channel: binding.channel };
  }
  const source = states.find(({ key }) => key === command.state)?.source;
  if (source?.type === "local") return { type: "local", state: command.state };
  if (source?.type === "channel" && source.command != null)
    return { type: "channel", channel: source.command };
  return undefined;
};

/** @returns each channel that a command of spec writes, once. */
export const commandChannels = (
  spec: Pick<schematic.symbol.Spec, "commands">,
  states: Sourced[],
  commands: schematic.CommandBinding[],
): channel.Key[] =>
  unique.unique(
    spec.commands.flatMap((command) => {
      const t = target(command, states, commands);
      return t?.type === "channel" ? [t.channel] : [];
    }),
  );

/** @returns a number sink for each channel in channels, by its key. */
export const sinks = (channels: channel.Key[]): Record<string, telem.NumberSinkSpec> =>
  Object.fromEntries(channels.map((key) => [key, Telem.numberSink(key)]));

/** @returns a number source for each channel in channels, by its key. */
export const sources = (
  channels: channel.Key[],
): Record<string, telem.NumberSourceSpec> =>
  Object.fromEntries(channels.map((ch) => [ch, Telem.numberSource(ch)]));

// Readings are keyed by channel.
const reading = (
  readings: Record<string, symbolNode.Reading>,
  key: channel.Key | undefined,
): symbolNode.Reading | undefined =>
  key != null && Object.hasOwn(readings, key) ? readings[key] : undefined;

const stateReading = (
  readings: Record<string, symbolNode.Reading>,
  source: schematic.StateBinding["source"],
): symbolNode.Reading | undefined =>
  reading(readings, source?.type === "channel" ? source.key : undefined);

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
      source?.type === "local" ? source.value : stateReading(readings, source)?.value;
    if (v != null) out.set(key, v);
  }
  return out;
};

/**
 * @returns the reading of each bound state that has a source, by its key. A state whose
 * channel has no data reads as NaN.
 */
export const stateReadings = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): Record<string, Custom.Reading> => {
  const out: Record<string, Custom.Reading> = {};
  for (const { key, source } of bindings)
    if (source?.type === "local") out[key] = { value: source.value };
    else if (source?.type === "channel" && source.key != null) {
      const r = stateReading(readings, source);
      out[key] = { value: r?.value ?? NaN, stale: r?.stale };
    }
  return out;
};

/** @returns whether a bound state reads a stale channel. */
export const stale = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): boolean =>
  bindings.some(({ source }) => stateReading(readings, source)?.stale === true);

/**
 * @returns the key of each state that reads a channel that has no data yet, or whose
 * newest value is NaN.
 */
export const missing = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): Set<string> =>
  new Set(
    bindings.flatMap(({ key, source }) =>
      source?.type === "channel" &&
      source.key != null &&
      stateReading(readings, source)?.value == null
        ? [key]
        : [],
    ),
  );

/** @returns the newest value on the command channel of each bound state, by its key. */
export const commanded = (
  bindings: Sourced[],
  readings: Record<string, symbolNode.Reading>,
): Map<string, number> => {
  const out = new Map<string, number>();
  for (const { key, source } of bindings) {
    if (source?.type !== "channel") continue;
    const v = reading(readings, source.command)?.value;
    if (v != null) out.set(key, v);
  }
  return out;
};
