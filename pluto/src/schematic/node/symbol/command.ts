// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, type schematic } from "@synnaxlabs/client";
import { unique } from "@synnaxlabs/x";

import { Custom } from "@/schematic/node/common/custom";
import { Telem } from "@/schematic/node/common/telem";
import type { telem } from "@/telem/aether";

type Bindings = Pick<schematic.SymbolNodeConfig, "states" | "commands">;

/** Target is where a command writes: a channel, or the local value of a state. */
export type Target =
  { type: "channel"; channel: channel.Key } | { type: "local"; state: string };

const find = <T extends { key: string }>(items: T[], key: string, kind: string): T => {
  const item = items.find((i) => i.key === key);
  if (item == null) throw new NotFoundError(`symbol has no ${kind} ${key}`);
  return item;
};

/** @returns where command writes, or undefined when its binding sets no channel. */
export const target = (
  command: schematic.symbol.Command,
  { states, commands }: Bindings,
): Target | undefined => {
  if (command.state == null) {
    const channel = commands.find((c) => c.key === command.key)?.channel;
    return channel == null ? undefined : { type: "channel", channel };
  }
  const source = states.find((s) => s.key === command.state)?.source;
  if (source?.type === "local") return { type: "local", state: command.state };
  if (source?.type === "channel" && source.command != null)
    return { type: "channel", channel: source.command };
  return undefined;
};

/** @returns the channels that the commands of spec write. */
export const commandChannels = (
  spec: schematic.symbol.Spec | undefined,
  bindings: Bindings,
): channel.Key[] =>
  unique.unique(
    (spec?.commands ?? []).flatMap((c) => {
      const t = target(c, bindings);
      return t?.type === "channel" ? [t.channel] : [];
    }),
  );

/** @returns a number sink for each channel in channels, by its key. */
export const sinks = (channels: channel.Key[]): Record<string, telem.NumberSinkSpec> =>
  Object.fromEntries(channels.map((ch) => [ch, Telem.numberSink(ch)]));

const commandKeys = (interaction: schematic.symbol.Interaction): string[] =>
  interaction.type === "fire" ? [interaction.command] : interaction.commands;

/**
 * @returns the interactions of spec that take clicks: each of their commands writes a
 * fixed value somewhere.
 */
export const clickable = (
  spec: schematic.symbol.Spec,
  bindings: Bindings,
): schematic.symbol.Interaction[] =>
  spec.interactions.filter((i) =>
    commandKeys(i).every((key) => {
      const command = find(spec.commands, key, "command");
      return command.value.type === "fixed" && target(command, bindings) != null;
    }),
  );

/**
 * @returns the interaction that a click on el takes: one whose part holds el, else one
 * on the whole drawing. root is the element that holds the drawing.
 */
export const interactionAt = (
  spec: schematic.symbol.Spec,
  interactions: schematic.symbol.Interaction[],
  root: Element,
  el: Element,
): schematic.symbol.Interaction | undefined =>
  interactions.find((i) => {
    if (i.part == null) return false;
    const { selectors } = find(spec.parts, i.part, "part");
    return selectors.some((s) =>
      Array.from(root.querySelectorAll(s)).some((shape) => shape.contains(el)),
    );
  }) ?? interactions.find((i) => i.part == null);

/**
 * @returns the value that command writes.
 * @throws {Error} if command asks for its value.
 */
export const fixedValue = (command: schematic.symbol.Command): number => {
  if (command.value.type !== "fixed") throw new Error(`command ${command.key} asks`);
  return command.value.value;
};

/**
 * @returns the command that a click on interaction sends, or undefined when the value
 * of a `next` state equals neither command. symbol matches the command values, and
 * bound, with the bindings of the node applied, matches the state value.
 */
export const pick = (
  symbol: schematic.symbol.Spec,
  bound: schematic.symbol.Spec,
  interaction: schematic.symbol.Interaction,
  value: ReadonlyMap<string, number>,
): schematic.symbol.Command | undefined => {
  if (interaction.type === "fire")
    return find(symbol.commands, interaction.command, "command");
  const [a, b] = interaction.commands.map((key) =>
    find(symbol.commands, key, "command"),
  );
  if (a.state == null) throw new NotFoundError(`next command ${a.key} has no state`);
  const v = value.get(a.state);
  if (v == null) return undefined;
  const matchedValue = Custom.matched(find(bound.states, a.state, "state"), v);
  const state = find(symbol.states, a.state, "state");
  if (matchedValue === Custom.matched(state, fixedValue(a))) return b;
  if (matchedValue === Custom.matched(state, fixedValue(b))) return a;
  return undefined;
};
