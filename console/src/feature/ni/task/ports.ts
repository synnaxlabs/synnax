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
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type CIChannel,
  type DIChannel,
  type DigitalChannel,
  type DOChannel,
  isGenerated,
  type TerminalConfig,
} from "@/feature/ni/task/types";
import { Task } from "@/platform/task";

/**
 * A port of a device, and the port a differential channel on it uses as its negative
 * input. A negative of 0 means that the port has no pair.
 */
export interface Port {
  port: number;
  negative: number;
}

/**
 * Whether a port accepts the value, given the list of values it accepts. An empty list
 * means DAQmx did not report the list, so the port accepts every value.
 */
export const accepts = (list: readonly string[], value: string): boolean =>
  list.length === 0 || list.includes(value);

/** A channel's hold on a port: its own port, or the negative input of its pair. */
export interface Claim extends Task.Port.Claim {
  negative: boolean;
}

/** The claims on each port of one device. */
export type Claims = Task.Port.Claims<Claim>;

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
  if (terminalConfig !== "Cfg_Default" || port == null) return terminalConfig;
  return port.defaultTerminalConfig;
};

/** The ports an analog input channel holds: its own, then its negative input. */
const analogInputHolds = (channel: AIChannel, hardware?: ni.Hardware): string[] => {
  if (!("port" in channel)) return [];
  const port = hardware?.analogInputs.find((p) => p.port === channel.port);
  if (
    port == null ||
    port.negative === 0 ||
    resolveTerminalConfig(channel, port) !== "Diff"
  )
    return [`ai${channel.port}`];
  return [`ai${channel.port}`, `ai${port.negative}`];
};

/**
 * The lowest port whose holds are all free. Without the device's ports, it counts up
 * from 0. When every port is taken, it takes the port after the last, which the checks
 * mark as missing.
 */
export const lowestFreePort = (
  ports: Port[] | null,
  claims: Claims,
  holds: (port: Port) => string[],
): number => {
  const free = (p: Port) => holds(p).every((h) => !claims.has(h));
  if (ports == null) {
    let port = 0;
    while (!free({ port, negative: 0 })) port++;
    return port;
  }
  const found = ports.find(free);
  if (found != null) return found.port;
  return ports.length === 0 ? 0 : Math.max(...ports.map(({ port }) => port)) + 1;
};

const numbered = (list: { port: number }[]): Port[] =>
  list.map(({ port }) => ({ port, negative: 0 }));

/** The digital ports that read input, or that write output. */
export const digitalPorts = (
  hardware: ni.Hardware,
  direction: "input" | "output",
): ni.DigitalPort[] => hardware.digitalPorts.filter((p) => p[direction]);

/** The DAQmx name of a digital line, such as "port0/line3". */
export const lineName = (port: number, line: number): string =>
  `port${port}/line${line}`;

/** The claims on each line of a digital task. */
export const lineClaims = (channels: DigitalChannel[]): Claims =>
  Task.Port.group(
    channels.map(({ key, port, line, disabled }) => ({
      port: lineName(port, line),
      channel: key,
      negative: false,
      disabled,
    })),
  );

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
}: GroupsProps): Task.Port.Group[] => {
  const name = (port: number) => `${prefix}${port}`;
  const on = (port: number) => claims.get(name(port)) ?? [];
  const cell = (
    port: number,
    row: number,
    column: number,
    negative: number = 0,
  ): Task.Port.Cell =>
    Task.Port.cell({
      key: String(port),
      labels: negative === 0 ? [String(port)] : [String(port), String(negative)],
      row,
      column,
      ports: negative === 0 ? [name(port)] : [name(port), name(negative)],
      claims,
      selected,
      describe,
    });
  const negatives = new Set(
    ports.map(({ negative }) => negative).filter((n) => n !== 0),
  );
  const positives = ports.filter(({ negative }) => negative !== 0);
  const result: Task.Port.Group[] = [];
  for (let start = 0; start < positives.length; start += PAIRED_GROUP) {
    const chunk = positives.slice(start, start + PAIRED_GROUP);
    const cells: Task.Port.Cell[] = [];
    chunk.forEach(({ port, negative }, i) => {
      const differential = on(port).some(({ channel }) =>
        on(negative).some((c) => c.negative && c.channel === channel),
      );
      if (differential) cells.push(cell(port, 1, i + 1, negative));
      else cells.push(cell(port, 1, i + 1), cell(negative, 2, i + 1));
    });
    result.push({ key: `pairs-${chunk[0].port}`, columns: chunk.length, cells });
  }
  const rest = ports.filter(
    ({ port, negative }) => negative === 0 && !negatives.has(port),
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

/** The ports of one kind, such as analog inputs, and how channels hold them. */
export interface PortKind<C extends Task.Channel> {
  /** The prefix of a port's DAQmx name: "ai", "ao", "ctr". */
  prefix: string;
  ports: (hardware: ni.Hardware) => Port[];
  /** The names of the ports the channel holds, its own port first. */
  holds: (channel: C, hardware?: ni.Hardware) => string[];
  /** The name a channel shows by. */
  name: (channel: C, device: Device) => string;
}

/** The claims of the channels, all on one device. */
export const claimsOf = <C extends Task.Channel>(
  kind: PortKind<C>,
  channels: C[],
  hardware?: ni.Hardware,
): Claims =>
  Task.Port.group(
    channels.flatMap((c) =>
      kind.holds(c, hardware).map((port, i) => ({
        port,
        channel: c.key,
        negative: i > 0,
        disabled: c.disabled,
      })),
    ),
  );

/**
 * The state of the ports the channel holds, given the channels on its device: in
 * conflict when another channel holds one.
 */
export const holdState = <C extends Task.Channel>(
  kind: PortKind<C>,
  channels: C[],
  channel: C,
  hardware?: ni.Hardware,
): Task.Port.State => {
  const claims = claimsOf(kind, channels, hardware);
  return Task.Port.stateOf(
    kind.holds(channel, hardware).map((p) => claims.get(p) ?? []),
  );
};

const identifierOf = (device: Device): string =>
  channel.escapeInvalidName(device.properties.identifier);

const named = (name: string, fallback: () => string): string =>
  primitive.isNonZero(name) ? name : fallback();

export const ANALOG_INPUT: PortKind<AIChannel> = {
  prefix: "ai",
  ports: (hardware) =>
    hardware.analogInputs.map(({ port, negative }) => ({ port, negative })),
  holds: analogInputHolds,
  name: (c, d) => named(c.name, () => `${identifierOf(d)}_ai_${getAIChannelSuffix(c)}`),
};

export const ANALOG_OUTPUT: PortKind<AOChannel> = {
  prefix: "ao",
  ports: (hardware) => numbered(hardware.analogOutputs),
  holds: ({ port }) => [`ao${port}`],
  name: (c, d) =>
    isGenerated(c)
      ? AO_CHANNEL_TYPE_NAMES[c.type]
      : named(c.cmdChannelName, () => `${identifierOf(d)}_ao_${c.port}_cmd`),
};

export const COUNTER_INPUT: PortKind<CIChannel> = {
  prefix: "ctr",
  ports: (hardware) => numbered(hardware.counterInputs),
  holds: ({ port }) => [`ctr${port}`],
  name: (c, d) => named(c.name, () => `${identifierOf(d)}_ctr_${c.port}`),
};

/** The name an analog output channel's state channel shows by. */
export const analogOutputStateName = (
  c: Exclude<AOChannel, { type: "ao_func_gen" }>,
  d: Device,
): string => named(c.stateChannelName, () => `${identifierOf(d)}_ao_${c.port}_state`);

/** The name a digital input channel shows by. */
export const digitalInputName = (
  c: Pick<DIChannel, "name" | "port" | "line">,
  d: Device,
): string => named(c.name, () => `${identifierOf(d)}_di_${c.port}_${c.line}`);

/** The name a digital output channel shows by: its command channel's. */
export const digitalOutputName = (
  c: Pick<DOChannel, "cmdChannelName" | "port" | "line">,
  d: Device,
): string =>
  named(c.cmdChannelName, () => `${identifierOf(d)}_do_${c.port}_${c.line}_cmd`);

/** The name of a digital output channel's state channel. */
export const digitalOutputStateName = (
  c: Pick<DOChannel, "stateChannelName" | "port" | "line">,
  d: Device,
): string =>
  named(c.stateChannelName, () => `${identifierOf(d)}_do_${c.port}_${c.line}_state`);
