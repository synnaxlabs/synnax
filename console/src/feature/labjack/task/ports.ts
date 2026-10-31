// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Model, type Port, PORTS } from "@/feature/labjack/device/types";
import { type Channel } from "@/feature/labjack/task/types";
import { type Task } from "@/platform/task";

/** The name on the port's terminal: its alias, such as "FIO4", or else its key. */
export const terminal = ({ key, alias }: Port): string => alias ?? key;

/**
 * The terminal of the port with the key on the model. Ports of two types can share a
 * terminal: on a T4, AIN4 and DIO4 are both FIO4. A port the model lacks is its own.
 */
export const terminalOf = (model: Model, key: string): string => {
  const lists: Port[][] = Object.values(PORTS[model]);
  for (const ports of lists) {
    const port = ports.find((p) => p.key === key);
    if (port != null) return terminal(port);
  }
  return key;
};

/** Ports in sequence on one terminal prefix, such as FIO4 to FIO7. */
export interface Bank {
  /** The terminal prefix and the accessory the ports need, such as "AIN (MUX80)". */
  title: string;
  ports: Port[];
}

const TERMINAL_REGEX = /^(\D+)(\d+)$/;

const split = (port: Port): [string, number] => {
  const name = terminal(port);
  const match = TERMINAL_REGEX.exec(name);
  if (match == null) throw new Error(`LabJack terminal ${name} has no number`);
  return [match[1], Number(match[2])];
};

/** Splits the ports into banks where the terminal prefix or the accessory changes. */
export const banks = (ports: Port[]): Bank[] => {
  const result: Bank[] = [];
  for (const port of ports) {
    const [prefix] = split(port);
    const { accessory } = port;
    const bank = result.at(-1);
    const last = bank?.ports.at(-1);
    if (bank != null && last != null) {
      const [lastPrefix] = split(last);
      if (lastPrefix === prefix && last.accessory === accessory) {
        bank.ports.push(port);
        continue;
      }
    }
    result.push({
      title: accessory == null ? prefix : `${prefix} (${accessory})`,
      ports: [port],
    });
  }
  return result;
};

/** A channel on a terminal. */
export interface Claim {
  channel: string;
  /** The Driver skips a disabled channel, so its hold never conflicts. */
  disabled: boolean;
}

/** The channels on each terminal. */
export type Claims = Map<string, Claim[]>;

export const claims = (channels: Channel[], model: Model): Claims => {
  const result: Claims = new Map();
  channels.forEach(({ key, port, disabled }) => {
    const t = terminalOf(model, port);
    result.set(t, [...(result.get(t) ?? []), { channel: key, disabled }]);
  });
  return result;
};

/** Whether more than one enabled channel holds the terminal. */
export const conflicts = (list: Claim[]): boolean =>
  list.filter((c) => !c.disabled).length > 1;

/** The channels, enabled ones first, each group in its order. */
export const enabledFirst = <C extends Channel>(channels: C[]): C[] =>
  channels.toSorted((a, b) => Number(a.disabled) - Number(b.disabled));

/** How many of the ports have a channel on their terminal. */
export const usedCount = (ports: Port[], claims: Claims): number =>
  ports.filter((p) => claims.has(terminal(p))).length;

export interface GroupsProps {
  bank: Bank;
  claims: Claims;
  selected: string[];
  /** Names the channel with the key, for a tooltip. */
  describe: (channel: string) => string;
}

const WIDTH = 8;

const GROUP = 16;

/** Lays a bank out as the grid shows it: eight ports to a row, sixteen to a group. */
export const groups = ({
  bank,
  claims,
  selected,
  describe,
}: GroupsProps): Task.PortGroup[] => {
  const cell = (port: Port, i: number, width: number): Task.PortCell => {
    const name = terminal(port);
    const list = claims.get(name) ?? [];
    const names = list
      .map((c) =>
        c.disabled ? `${describe(c.channel)} (disabled)` : describe(c.channel),
      )
      .join(" and ");
    let tooltip = `${name} is free`;
    if (list.length === 1) tooltip = `${name}: ${names}`;
    else if (list.length > 1) tooltip = `${name} is used by ${names}`;
    return {
      key: port.key,
      name,
      labels: [String(split(port)[1])],
      row: Math.floor(i / width) + 1,
      column: (i % width) + 1,
      state: list.length === 0 ? "free" : conflicts(list) ? "conflict" : "used",
      disabled: list.length > 0 && list.every((c) => c.disabled),
      selected: list.some((c) => selected.includes(c.channel)),
      channel: (list.find((c) => !c.disabled) ?? list[0])?.channel,
      tooltip,
    };
  };
  const result: Task.PortGroup[] = [];
  for (let start = 0; start < bank.ports.length; start += GROUP) {
    const chunk = bank.ports.slice(start, start + GROUP);
    const width = Math.min(WIDTH, chunk.length);
    result.push({
      key: chunk[0].key,
      columns: width,
      cells: chunk.map((port, i) => cell(port, i, width)),
    });
  }
  return result;
};

/** Names the channel with the key for a tooltip, by the name it shows by. */
export const describer =
  <C extends Channel>(channels: C[], name: (channel: C) => string) =>
  (key: string): string => {
    const channel = channels.find((c) => c.key === key);
    return channel == null ? "" : name(channel);
  };
