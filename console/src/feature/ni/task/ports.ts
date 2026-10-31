// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type ni } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";

import { type Device } from "@/feature/ni/device/types";
import { getAIChannelSuffix } from "@/feature/ni/task/getAIChannelDeviceKey";
import {
  type AIChannel,
  type AOChannel,
  type CIChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
  type TerminalConfig,
} from "@/feature/ni/task/types";
import { type Task } from "@/platform/task";

/**
 * A port of a device, and the port a differential channel on it uses as its negative
 * input.
 */
export interface Port {
  port: number;
  negative?: number;
}

/**
 * Whether a port accepts the value, given the list of values it accepts. An empty list
 * means DAQmx did not report the list, so the port accepts every value.
 */
export const accepts = (list: readonly string[], value: string): boolean =>
  list.length === 0 || list.includes(value);

/** A channel's hold on a port: its own port, or the negative input of its pair. */
export interface Claim {
  channel: string;
  negative: boolean;
  /** The Driver skips a disabled channel, so its hold never conflicts. */
  disabled: boolean;
}

/** The channels that hold each port of one device. */
export type Claims = Map<number, Claim[]>;

const enabled = (list: Claim[]): Claim[] => list.filter((c) => !c.disabled);

/** Whether more than one enabled channel holds the port. */
export const conflicts = (list: Claim[]): boolean => enabled(list).length > 1;

/** The channels, enabled ones first, each group in its order. */
export const enabledFirst = <C extends Task.Channel>(channels: C[]): C[] =>
  channels.toSorted((a, b) => Number(a.disabled) - Number(b.disabled));

const claim = (claims: Claims, port: number, c: Claim): void => {
  const list = claims.get(port);
  if (list == null) claims.set(port, [c]);
  else list.push(c);
};

/**
 * The terminal configuration an analog input channel gets on its port, with "Default"
 * resolved to what the port gives. Null for a channel type that has none.
 */
const resolveTerminalConfig = (
  channel: AIChannel,
  port?: ni.AnalogInputPort,
): TerminalConfig | null => {
  if (!("terminalConfig" in channel)) return null;
  const { terminalConfig } = channel;
  if (terminalConfig !== "Cfg_Default") return terminalConfig;
  return port?.defaultTerminalConfig ?? terminalConfig;
};

/** The ports an analog input channel holds: its own, then its negative input. */
export const analogInputHolds = (
  channel: AIChannel,
  hardware?: ni.Hardware,
): number[] => {
  if (!("port" in channel)) return [];
  const port = hardware?.analogInputs.find((p) => p.port === channel.port);
  if (port?.negative == null || resolveTerminalConfig(channel, port) !== "Diff")
    return [channel.port];
  return [channel.port, port.negative];
};

/** The channels on each analog input port of the device. */
export const analogInputClaims = (
  channels: AIChannel[],
  device: string,
  hardware?: ni.Hardware,
): Claims => {
  const claims: Claims = new Map();
  channels.forEach((c) => {
    if (c.device !== device) return;
    analogInputHolds(c, hardware).forEach((port, i) =>
      claim(claims, port, { channel: c.key, negative: i > 0, disabled: c.disabled }),
    );
  });
  return claims;
};

/** The channels on each port, for channel types that hold one port each. */
export const portClaims = (channels: (Task.Channel & { port: number })[]): Claims => {
  const claims: Claims = new Map();
  channels.forEach(({ key, port, disabled }) =>
    claim(claims, port, { channel: key, negative: false, disabled }),
  );
  return claims;
};

/**
 * The lowest port whose holds are all free. Without the device's ports, it counts up
 * from 0. When every port is taken, it takes the port after the last, which the checks
 * mark as missing.
 */
export const lowestFreePort = (
  ports: Port[] | null,
  claims: Claims,
  holds: (port: Port) => number[] = ({ port }) => [port],
): number => {
  const free = (p: Port) => holds(p).every((h) => !claims.has(h));
  if (ports == null) {
    let port = 0;
    while (!free({ port })) port++;
    return port;
  }
  const found = ports.find(free);
  if (found != null) return found.port;
  return ports.length === 0 ? 0 : Math.max(...ports.map(({ port }) => port)) + 1;
};

/** The analog input ports of the device, as the grid lays them out. */
export const analogInputPorts = (hardware: ni.Hardware): Port[] =>
  hardware.analogInputs.map(({ port, negative }) => ({ port, negative }));

const numbered = (list: { port: number }[]): Port[] =>
  list.map(({ port }) => ({ port }));

export const analogOutputPorts = (hardware: ni.Hardware): Port[] =>
  numbered(hardware.analogOutputs);

export const counterInputPorts = (hardware: ni.Hardware): Port[] =>
  numbered(hardware.counterInputs);

/** The digital ports that read input, or that write output. */
export const digitalPorts = (
  hardware: ni.Hardware,
  direction: "input" | "output",
): ni.DigitalPort[] => hardware.digitalPorts.filter((p) => p[direction]);

/** Names a digital line in the form a task stores it. */
export const lineKey = (port: number, line: number): string => `${port}/${line}`;

/** The channels on each line of a digital task. */
export const lineClaims = (channels: DigitalChannel[]): Map<string, Claim[]> => {
  const claims = new Map<string, Claim[]>();
  channels.forEach(({ key, port, line, disabled }) => {
    const k = lineKey(port, line);
    claims.set(k, [
      ...(claims.get(k) ?? []),
      { channel: key, negative: false, disabled },
    ]);
  });
  return claims;
};

export interface GroupsProps {
  ports: Port[];
  claims: Claims;
  /** The prefix of a port's DAQmx name: "ai", "ao", "ctr". */
  prefix: string;
  selected: string[];
  /** Names the channel with the key, for a tooltip. */
  describe: (channel: string) => string;
}

const UNPAIRED_WIDTH = 8;

const UNPAIRED_GROUP = 16;

const PAIRED_GROUP = 8;

const stateOf = (lists: Claim[][]): Task.PortState => {
  if (lists.some(conflicts)) return "conflict";
  return lists.some((l) => l.length > 0) ? "used" : "free";
};

// An enabled channel owns a cell before a disabled one, and a port before the negative
// input of its pair.
const rank = ({ disabled, negative }: Claim): number =>
  Number(disabled) * 2 + Number(negative);

export interface Hold {
  state: Task.PortState;
  /** Only disabled channels hold the cell. */
  disabled: boolean;
  /** The channel a click on the cell selects. */
  channel?: string;
  /** The channels on the cell, such as "chan_a and chan_b (disabled)". */
  names: string;
}

/** How a cell shows the channels on its ports. */
export const holdOf = (
  lists: Claim[][],
  describe: (channel: string) => string,
): Hold => {
  const all = lists.flat();
  const unique = [...new Map(all.map((c) => [c.channel, c])).values()];
  return {
    state: stateOf(lists),
    disabled: all.length > 0 && all.every((c) => c.disabled),
    channel: all.toSorted((a, b) => rank(a) - rank(b))[0]?.channel,
    names: unique
      .map((c) =>
        c.disabled ? `${describe(c.channel)} (disabled)` : describe(c.channel),
      )
      .join(" and "),
  };
};

/**
 * Lays the ports out as the grid shows them. A port with a negative input sits above
 * it, eight pairs to a group; a differential channel on the pair spans both. Other
 * ports sit eight to a row, sixteen to a group.
 */
export const groups = ({
  ports,
  claims,
  prefix,
  selected,
  describe,
}: GroupsProps): Task.PortGroup[] => {
  const name = (port: number) => `${prefix}${port}`;
  const tooltip = (port: number, lists: Claim[][], negative?: number): string => {
    const all = lists.flat();
    if (all.length === 0) return `${name(port)} is free`;
    const { names, state } = holdOf(lists, describe);
    if (negative != null) {
      const pair = `${name(port)} and ${name(negative)}`;
      if (state === "conflict") return `${pair} are used by ${names}`;
      return `${pair}: ${names}`;
    }
    if (all.length > 1) return `${name(port)} is used by ${names}`;
    return `${name(port)}: ${names}`;
  };
  const cell = (
    port: number,
    row: number,
    column: number,
    negative?: number,
  ): Task.PortCell => {
    const lists = [claims.get(port) ?? []];
    if (negative != null) lists.push(claims.get(negative) ?? []);
    const paired = negative != null;
    const { state, disabled, channel } = holdOf(lists, describe);
    return {
      key: String(port),
      name: name(port),
      labels: paired ? [String(port), String(negative)] : [String(port)],
      row,
      column,
      paired,
      state,
      disabled,
      selected: lists.flat().some((c) => selected.includes(c.channel)),
      channel,
      tooltip: tooltip(port, lists, negative),
    };
  };
  const negatives = new Set(
    ports.map(({ negative }) => negative).filter((n) => n != null),
  );
  const positives = ports.filter(({ negative }) => negative != null);
  const result: Task.PortGroup[] = [];
  for (let start = 0; start < positives.length; start += PAIRED_GROUP) {
    const chunk = positives.slice(start, start + PAIRED_GROUP);
    const cells: Task.PortCell[] = [];
    chunk.forEach(({ port, negative }, i) => {
      const n = negative as number;
      const differential = (claims.get(port) ?? []).some(({ channel }) =>
        claims.get(n)?.some((c) => c.negative && c.channel === channel),
      );
      if (differential) cells.push(cell(port, 1, i + 1, n));
      else cells.push(cell(port, 1, i + 1), cell(n, 2, i + 1));
    });
    result.push({ key: `pairs-${chunk[0].port}`, columns: chunk.length, cells });
  }
  const rest = ports.filter(
    ({ port, negative }) => negative == null && !negatives.has(port),
  );
  for (let start = 0; start < rest.length; start += UNPAIRED_GROUP) {
    const chunk = rest.slice(start, start + UNPAIRED_GROUP);
    const width = Math.min(UNPAIRED_WIDTH, chunk.length);
    result.push({
      key: `ports-${chunk[0].port}`,
      columns: width,
      cells: chunk.map(({ port }, i) =>
        cell(port, Math.floor(i / width) + 1, (i % width) + 1),
      ),
    });
  }
  return result;
};

/** How many of the device's ports hold a channel. */
export const usedCount = (ports: Port[], claims: Claims): number =>
  ports.filter(({ port }) => claims.has(port)).length;

/** The ports of one kind, such as analog inputs, and how channels hold them. */
export interface PortKind<C extends Task.Channel> {
  /** The prefix of a port's DAQmx name: "ai", "ao", "ctr". */
  prefix: string;
  ports: (hardware: ni.Hardware) => Port[];
  claims: (channels: C[], device: string, hardware?: ni.Hardware) => Claims;
  /** The name a channel shows by. */
  name: (channel: C, device: Device) => string;
}

/** Names the channel with the key for a tooltip, by the kind's names. */
export const describer =
  <C extends Task.Channel>(kind: PortKind<C>, channels: C[], device: Device) =>
  (key: string): string => {
    const channel = channels.find((c) => c.key === key);
    return channel == null ? "" : kind.name(channel, device);
  };

const identifierOf = (device: Device): string =>
  channel.escapeInvalidName(device.properties.identifier);

const named = (name: string, fallback: () => string): string =>
  primitive.isNonZero(name) ? name : fallback();

export const ANALOG_INPUT: PortKind<AIChannel> = {
  prefix: "ai",
  ports: analogInputPorts,
  claims: analogInputClaims,
  name: (c, d) => named(c.name, () => `${identifierOf(d)}_ai_${getAIChannelSuffix(c)}`),
};

export const ANALOG_OUTPUT: PortKind<AOChannel> = {
  prefix: "ao",
  ports: analogOutputPorts,
  claims: (channels) => portClaims(channels),
  name: (c, d) => named(c.cmdChannelName, () => `${identifierOf(d)}_ao_${c.port}_cmd`),
};

export const COUNTER_INPUT: PortKind<CIChannel> = {
  prefix: "ctr",
  ports: counterInputPorts,
  claims: (channels, device) => portClaims(channels.filter((c) => c.device === device)),
  name: (c, d) => named(c.name, () => `${identifierOf(d)}_ctr_${c.port}`),
};

/** The name an analog output channel's state channel shows by. */
export const analogOutputStateName = (c: AOChannel, d: Device): string =>
  named(c.stateChannelName, () => `${identifierOf(d)}_ao_${c.port}_state`);

/** The name a digital input channel shows by. */
export const digitalInputName = (c: DIChannel, d: Device): string =>
  named(c.name, () => `${identifierOf(d)}_di_${c.port}_${c.line}`);

/** The name a digital output channel shows by: its command channel's. */
export const digitalOutputName = (c: DOChannel, d: Device): string =>
  named(c.cmdChannelName, () => `${identifierOf(d)}_do_${c.port}_${c.line}_cmd`);

/** The state of a channel's ports: in conflict when another channel holds one. */
export const holdState = (claims: Claims, holds: number[]): Task.PortState =>
  holds.some((h) => conflicts(claims.get(h) ?? [])) ? "conflict" : "used";
