// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/feature/labjack/device";
import { type Channel } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

/** The name on the port's terminal: its alias, such as "FIO4", or else its key. */
export const terminal = ({ key, alias }: Device.Port): string => alias ?? key;

/**
 * The terminal of the port with the key on the model. Ports of two types can share a
 * terminal: on a T4, AIN4 and DIO4 are both FIO4. A port the model lacks is its own.
 */
export const terminalOf = (model: Device.Model, key: string): string => {
  const lists: Device.Port[][] = Object.values(Device.PORTS[model]);
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
  ports: Device.Port[];
}

const TERMINAL_REGEX = /^(\D+)(\d+)$/;

const split = (port: Device.Port): [string, number] => {
  const name = terminal(port);
  const match = TERMINAL_REGEX.exec(name);
  if (match == null) throw new Error(`LabJack terminal ${name} has no number`);
  return [match[1], Number(match[2])];
};

/** Splits the ports into banks where the terminal prefix or the accessory changes. */
export const banks = (ports: Device.Port[]): Bank[] => {
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

/** The channels on each terminal of the model. */
export const claims = (channels: Channel[], model: Device.Model): Task.Port.Claims =>
  Task.Port.group(
    channels.map(({ key, port, disabled }) => ({
      port: terminalOf(model, port),
      channel: key,
      disabled,
    })),
  );

export interface GroupsProps {
  bank: Bank;
  claims: Task.Port.Claims;
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
}: GroupsProps): Task.Port.Group[] => {
  const result: Task.Port.Group[] = [];
  for (let start = 0; start < bank.ports.length; start += GROUP) {
    const chunk = bank.ports.slice(start, start + GROUP);
    const width = Math.min(WIDTH, chunk.length);
    result.push({
      key: chunk[0].key,
      columns: width,
      cells: chunk.map((port, i) =>
        Task.Port.cell({
          key: port.key,
          labels: [String(split(port)[1])],
          row: Math.floor(i / width) + 1,
          column: (i % width) + 1,
          ports: [terminal(port)],
          claims,
          selected,
          describe,
        }),
      ),
    });
  }
  return result;
};
