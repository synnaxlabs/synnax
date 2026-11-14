// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { DataType, unique } from "@synnaxlabs/x";
import { type z } from "zod";

import { check, fill, findInput } from "@/feature/ethercat/device/settings";
import { canSwitch, isShared, of, onlyReaderOf } from "@/feature/ethercat/device/setup";
import {
  type SlaveDevice,
  type SlaveProperties,
} from "@/feature/ethercat/device/types";
import {
  addressKey,
  type Channel,
  DIRECTIONS,
  entryAt,
  type EntryChannel,
  getUnresolvedMessage,
  hasEntry,
  type PDOType,
} from "@/feature/ethercat/task/types";
import { Task } from "@/platform/task";

/** @returns the size in bytes of the number type that holds an entry. */
const entryWidth = (bitLength: number): number => {
  if (bitLength <= 8) return 1;
  if (bitLength <= 16) return 2;
  if (bitLength <= 32) return 4;
  return 8;
};

/** @returns what the user must do before Synnax can read an entry of the slave. */
const onlyReaderStep = (props: SlaveProperties): string => {
  if (of(props).kind === "switched") return "Restart the power of the slave.";
  return canSwitch(props)
    ? "In Configure, switch the slave to Synnax."
    : "Pick another entry.";
};

/**
 * @returns why the task cannot start with the enabled channel on the slave, or null.
 * channels holds the enabled channels of the task on the slave, ch among them.
 */
const entryProblem = (
  slave: SlaveDevice,
  channels: Channel[],
  ch: Channel,
  pdoType: PDOType,
): string | null => {
  if (ch.type === "unresolved") return getUnresolvedMessage(ch.pdo, slave, pdoType);
  if (!hasEntry(ch))
    return `A channel on slave ${slave.name} has no entry. Pick an entry.`;
  const address = ethercat.formatAddress(ch);
  const entry = entryAt(slave.properties.pdos[pdoType], ch);
  if (entry == null)
    return (
      `Slave ${slave.name} has no ${DIRECTIONS[pdoType]} entry at ${address}. ` +
      "Pick an entry."
    );
  const reader = pdoType === "inputs" ? onlyReaderOf(slave.properties, ch) : null;
  if (reader != null)
    return (
      `Entry ${address} on slave ${slave.name} sends data that only ${reader} ` +
      `reads. ${onlyReaderStep(slave.properties)}`
    );
  const dataType = new DataType(ch.dataType);
  const width = entryWidth(entry.bitLength);
  if (!(dataType.isInteger || dataType.isFloat) || dataType.density.valueOf() !== width)
    return (
      `Data type ${ch.dataType} does not fit entry ${address} on slave ` +
      `${slave.name}, which holds ${entry.bitLength} bits. Pick a ${width} ` +
      "byte number type."
    );
  const key = addressKey(ch);
  if (channels.some((o) => o.key !== ch.key && hasEntry(o) && addressKey(o) === key))
    return (
      `More than one channel uses entry ${address} on slave ${slave.name}. ` +
      "Pick a different entry for one of them."
    );
  return null;
};

/**
 * @returns why the task cannot start with the settings of the input that the channel
 * reads, or null. held holds the startup parameters of the task.
 */
const settingProblem = (
  slave: SlaveDevice,
  ch: EntryChannel,
  held: ethercat.SlaveStartupParameters[],
): string | null => {
  if (isShared(slave.properties)) return null;
  const input = findInput(slave.properties.settings, ch);
  if (input == null) return null;
  const prior = held.find((s) => s.device === slave.key)?.parameters ?? [];
  const [problem] = check(input, fill(input.settings, prior));
  if (problem == null) return null;
  const address = ethercat.formatAddress(ch);
  return `Entry ${address} on slave ${slave.name}: ${problem.message}`;
};

/**
 * @returns why the slave cannot start because another device on the rack is at its
 * network and position, or null. The problem names the device that is not connected,
 * which the user deletes.
 */
const positionProblem = (slave: SlaveDevice, onRack: SlaveDevice[]): string | null => {
  const { network, position } = slave.properties;
  const other = onRack.find(
    (o) =>
      o.key !== slave.key &&
      o.properties.network === network &&
      o.properties.position === position,
  );
  if (other == null) return null;
  const connected = (d: SlaveDevice): boolean =>
    onRack.find((o) => o.key === d.key)?.status?.details.connected === true;
  const stale = connected(slave) ? other : connected(other) ? slave : null;
  const step =
    stale == null
      ? "Delete the one that is not on the network."
      : `Delete ${stale.name}, which is not connected.`;
  return (
    `Slaves ${slave.name} and ${other.name} are both at position ${position} on ` +
    `${network}. ${step}`
  );
};

/** @returns why the slaves of the task cannot run on one network, or null. */
const networkProblem = (slaves: SlaveDevice[]): string | null => {
  const networks = unique.unique(slaves.map((s) => s.properties.network)).sort();
  if (networks.length > 1)
    return (
      `All slaves must be on the same network. Found: ${networks.join(", ")}. ` +
      "Pick slaves on one network."
    );
  if (networks.length === 1 && networks[0] === "")
    return "No network found for the selected slaves. Scan the network again.";
  return null;
};

/** The parts of the config of an EtherCAT task that its checks read. */
export interface CheckedConfig {
  channels: Channel[];
  /** The settings that a read task writes. A write task has none. */
  startupParameters?: ethercat.SlaveStartupParameters[];
}

interface Problem {
  path: PropertyKey[];
  message: string;
}

/**
 * @returns why the task cannot start on its slaves: task-wide problems at the channel
 * list, and each problem of an enabled channel at that channel. A channel whose slave
 * is not in slaves is not checked.
 */
const problemsOf = (
  pdoType: PDOType,
  { channels, startupParameters = [] }: CheckedConfig,
  slaves: SlaveDevice[],
  onRack: SlaveDevice[],
): Problem[] => {
  const used = slaves.filter((s) => channels.some((ch) => ch.device === s.key));
  const taskWide = [
    networkProblem(used),
    ...used.map((slave) => positionProblem(slave, onRack)),
  ].filter((p): p is string => p != null);
  const problems: Problem[] =
    taskWide.length === 0 ? [] : [{ path: ["channels"], message: taskWide.join(" ") }];
  const enabled = Map.groupBy(
    channels.filter((ch) => !ch.disabled),
    (ch) => ch.device,
  );
  channels.forEach((ch, i) => {
    const slave = used.find((s) => s.key === ch.device);
    if (slave == null || ch.disabled) return;
    let message = entryProblem(slave, enabled.get(ch.device) ?? [], ch, pdoType);
    if (message == null && pdoType === "inputs" && hasEntry(ch))
      message = settingProblem(slave, ch, startupParameters);
    if (message != null) problems.push({ path: ["channels", i], message });
  });
  return problems;
};

/**
 * Checks the channels of a task against their slaves as the user edits: one network for
 * every slave, no other device at the position of a slave, and for each enabled channel
 * its entry and, on inputs, the settings of the input it reads. A channel whose slave
 * has not loaded is not checked, so a deploy runs {@link checkSlaves} too.
 * @param slaves - The slaves that the channels of the task use.
 * @param onRack - The slaves on the rack of the task, with their statuses.
 */
export const slavesZ = (
  pdoType: PDOType,
  slaves: SlaveDevice[],
  onRack: SlaveDevice[],
): z.ZodType =>
  Task.check<CheckedConfig>((config, report) =>
    problemsOf(pdoType, config, slaves, onRack).forEach(({ path, message }) =>
      report(path, message),
    ),
  );

/**
 * Runs the checks of {@link slavesZ} on the slaves.
 * @throws {Error} with the first problem.
 */
export const checkSlaves = (
  pdoType: PDOType,
  config: CheckedConfig,
  slaves: SlaveDevice[],
  onRack: SlaveDevice[],
): void => {
  const [problem] = problemsOf(pdoType, config, slaves, onRack);
  if (problem != null) throw new Error(problem.message);
};
