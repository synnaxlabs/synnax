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
import { Task } from "@/platform/task";

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

/** The length of the model's longest terminal name, such as 6 for AIN127 on a T7. */
export const maxTerminalChars = (model: Model): number =>
  Math.max(
    ...Object.values(PORTS[model]).flatMap((ports: Port[]) =>
      ports.map((p) => terminal(p).length),
    ),
  );

/** Ports in sequence on one terminal prefix and connector, such as FIO4 to FIO7. */
export interface Bank {
  /** The terminal prefix, and the connector the ports need, such as "AIN (Mux80 X3)". */
  title: string;
  /** The connector the ports need, such as "X3", or else the terminal prefix. */
  name: string;
  ports: Port[];
}

const TERMINAL_REGEX = /^(\D+)(\d+)$/;

const split = (port: Port): [string, number] => {
  const name = terminal(port);
  const match = TERMINAL_REGEX.exec(name);
  if (match == null) throw new Error(`LabJack terminal ${name} has no number`);
  return [match[1], Number(match[2])];
};

const connectorOf = ({ connector }: Port): string | null =>
  connector == null ? null : `${connector.accessory} ${connector.name}`;

/** Splits the ports into banks where the terminal prefix or the connector changes. */
export const banks = (ports: Port[]): Bank[] => {
  const result: Bank[] = [];
  for (const port of ports) {
    const [prefix] = split(port);
    const connector = connectorOf(port);
    const bank = result.at(-1);
    const last = bank?.ports.at(-1);
    if (bank != null && last != null) {
      const [lastPrefix] = split(last);
      if (lastPrefix === prefix && connectorOf(last) === connector) {
        bank.ports.push(port);
        continue;
      }
    }
    result.push({
      title: connector == null ? prefix : `${prefix} (${connector})`,
      name: port.connector?.name ?? prefix,
      ports: [port],
    });
  }
  return result;
};

/** The numbers of the bank's first and last ports, such as "48–71". */
export const numbers = ({ ports }: Bank): string =>
  `${split(ports[0])[1]}–${split(ports[ports.length - 1])[1]}`;

/** The channels on each terminal of the model. */
export const claims = (channels: Channel[], model: Model): Task.Port.Claims =>
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

/** Lays a bank out as the grid shows it: eight ports to a row. */
export const groups = ({
  bank,
  claims,
  selected,
  describe,
}: GroupsProps): Task.Port.Group[] => {
  const width = Math.min(WIDTH, bank.ports.length);
  return [
    {
      key: bank.ports[0].key,
      columns: width,
      cells: bank.ports.map((port, i) =>
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
    },
  ];
};
