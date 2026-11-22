// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { compare, DataType, unique } from "@synnaxlabs/x";
import type { z } from "zod";

import { clashProblem, findClash } from "@/feature/ethercat/device/identifier";
import { check, fill, findInput } from "@/feature/ethercat/device/settings";
import { canSwitch, isShared, of, onlyReaderOf } from "@/feature/ethercat/device/setup";
import type {
  SlaveDevice,
  SlaveOnRack,
  SlaveProperties,
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
  if (of(props).kind === "switched") return "Restart the power of the module.";
  return canSwitch(props)
    ? "In Configure, switch the module to Synnax."
    : "Pick another entry.";
};

/**
 * @returns why the task cannot reach the slave, or null. A scan gives each slave it
 * finds a network, so a slave with none was never found, such as one made in code.
 */
const networkProblem = (slave: SlaveDevice): string | null =>
  slave.properties.network === ""
    ? `The Driver has never found ${slave.name} on a network. Select a connected ` +
      `device for each channel that uses ${slave.name}.`
    : null;

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
  if (!hasEntry(ch)) return null;
  const address = ethercat.formatAddress(ch);
  const entry = entryAt(slave.properties.pdos[pdoType], ch);
  if (entry == null)
    return (
      `Module ${slave.name} has no ${DIRECTIONS[pdoType]} entry at ${address}. ` +
      "Pick an entry."
    );
  const reader = pdoType === "inputs" ? onlyReaderOf(slave.properties, ch) : null;
  if (reader != null)
    return (
      `Entry ${address} on module ${slave.name} sends data that only ${reader} ` +
      `reads. ${onlyReaderStep(slave.properties)}`
    );
  const dataType = new DataType(ch.dataType);
  const width = entryWidth(entry.bitLength);
  if (!(dataType.isInteger || dataType.isFloat) || dataType.density.valueOf() !== width)
    return (
      `Data type ${ch.dataType} does not fit entry ${address} on module ` +
      `${slave.name}, which holds ${entry.bitLength} bits. Pick a ${width} ` +
      "byte number type."
    );
  const key = addressKey(ch);
  if (channels.some((o) => o.key !== ch.key && hasEntry(o) && addressKey(o) === key))
    return (
      `More than one channel uses entry ${address} on module ${slave.name}. ` +
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
  const [problem] = check(input, fill(input.settings, prior).values);
  if (problem == null) return null;
  const address = ethercat.formatAddress(ch);
  return `Entry ${address} on module ${slave.name}: ${problem.message}`;
};

type Slave = SlaveDevice | SlaveOnRack;

const order = compare.chain<Slave>(
  compare.by((s) => s.name, compare.natural),
  compare.by((s) => s.properties.position, compare.numbers),
  compare.by((s) => s.key, compare.codePoints),
);

/**
 * @returns the two slaves by name, then by slot, then by key, so that a problem of a
 * pair reads the same on the channels of either slave.
 */
const inOrder = (a: Slave, b: Slave): [Slave, Slave] =>
  order(a, b) <= 0 ? [a, b] : [b, a];

/** @returns the step that tells the user which of the two devices to delete. */
const deleteStep = (
  slave: SlaveDevice,
  other: SlaveOnRack,
  onRack: SlaveOnRack[],
): string => {
  const connected = ({ key }: { key: string }): boolean =>
    onRack.find((o) => o.key === key)?.connected === true;
  const slaveConnected = connected(slave);
  if (slaveConnected === connected(other))
    return "Delete the one that is not on the network.";
  const stale = slaveConnected ? other : slave;
  return `Delete ${stale.name}, which is not connected.`;
};

/**
 * @returns why the slave cannot start because another slave on the rack is at its
 * network and position, or null. The Driver binds a slave by its serial number, so the
 * problem holds only when the task uses both slaves or neither has a serial number. The
 * problem names the device that is not connected, which the user deletes.
 */
const positionProblem = (
  slave: SlaveDevice,
  used: SlaveDevice[],
  onRack: SlaveOnRack[],
): string | null => {
  const { network, position, serial } = slave.properties;
  const other = onRack.find(
    (o) =>
      o.key !== slave.key &&
      o.properties.network === network &&
      o.properties.position === position &&
      (used.some(({ key }) => key === o.key) ||
        (serial === 0 && o.properties.serial === 0)),
  );
  if (other == null) return null;
  const [first, second] = inOrder(slave, other);
  return (
    `Modules ${first.name} and ${second.name} are both at position ${position} on ` +
    `${network}. ${deleteStep(slave, other, onRack)}`
  );
};

/**
 * @returns why the slave cannot start because another device on its network is the same
 * module: the same vendor, product, and serial number. The problem names the device
 * that is not connected, which the user deletes. A slave with no serial number has no
 * problem, as the Driver cannot tell its module apart. The Driver keeps a serial number
 * unique on one network only, so two networks can each hold a module with it.
 */
const serialProblem = (slave: SlaveDevice, onRack: SlaveOnRack[]): string | null => {
  const { network, vendorId, productCode, serial } = slave.properties;
  if (serial === 0) return null;
  const other = onRack.find(
    ({ key, properties: p }) =>
      key !== slave.key &&
      p.network === network &&
      p.vendorId === vendorId &&
      p.productCode === productCode &&
      p.serial === serial,
  );
  if (other == null) return null;
  const [first, second] = inOrder(slave, other);
  return (
    `Modules ${first.name} and ${second.name} have the same serial number ${serial}. ` +
    `${deleteStep(slave, other, onRack)}`
  );
};

/** @returns why the slave cannot start while another slave has its identifier. */
const identifierProblem = (
  slave: SlaveDevice,
  onRack: SlaveOnRack[],
): string | null => {
  const other = findClash(slave, onRack);
  if (other == null) return null;
  const [first, second] = inOrder(slave, other);
  return clashProblem(second, first);
};

/** @returns why the slaves of the task cannot run on one network, or null. */
const mixedNetworksProblem = (slaves: SlaveDevice[]): string | null => {
  const networks = unique
    .unique(slaves.map((s) => s.properties.network).filter((n) => n !== ""))
    .sort();
  if (networks.length < 2) return null;
  return (
    `All modules must be on the same network. Found: ${networks.join(", ")}. ` +
    "Pick modules on one network."
  );
};

/** @returns why the task cannot start on the slave of a channel, or null. */
const deviceProblem = (
  slave: SlaveDevice,
  used: SlaveDevice[],
  onRack: SlaveOnRack[],
): string | null => {
  if (!slave.configured)
    return `${slave.name} is not configured. Configure it to start the task.`;
  return (
    networkProblem(slave) ??
    positionProblem(slave, used, onRack) ??
    serialProblem(slave, onRack) ??
    identifierProblem(slave, onRack)
  );
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
 * @returns why the task cannot start on its slaves: slaves on more than one network at
 * the channel list, a problem of a slave at the device of each of its channels, and
 * each other problem of an enabled channel at that channel. A channel with no device or
 * no entry, or whose slave is not in slaves, has no problem here.
 */
const problemsOf = (
  pdoType: PDOType,
  { channels, startupParameters = [] }: CheckedConfig,
  slaves: SlaveDevice[],
  onRack: SlaveOnRack[],
): Problem[] => {
  const used = slaves.filter((s) => channels.some((ch) => ch.device === s.key));
  const mixed = mixedNetworksProblem(used);
  const enabled = Map.groupBy(
    channels.filter((ch) => !ch.disabled),
    (ch) => ch.device,
  );
  const problems: Problem[] =
    mixed == null ? [] : [{ path: ["channels"], message: mixed }];
  channels.forEach((ch, i) => {
    const slave = used.find((s) => s.key === ch.device);
    if (slave == null) return;
    let message = deviceProblem(slave, used, onRack);
    if (message != null) {
      problems.push({ path: ["channels", i, "device"], message });
      return;
    }
    if (ch.disabled) return;
    message = entryProblem(slave, enabled.get(ch.device) ?? [], ch, pdoType);
    if (message == null && pdoType === "inputs" && hasEntry(ch))
      message = settingProblem(slave, ch, startupParameters);
    if (message != null) problems.push({ path: ["channels", i], message });
  });
  return problems;
};

/**
 * Checks the channels of a task against their slaves as the user edits: one network for
 * every slave, no other slave with the position, serial number, or identifier of a
 * slave, a configured slave that a scan found for each channel, and for each enabled
 * channel its entry and, on inputs, the settings of the input it reads. A channel whose
 * slave has not loaded is not checked, so a deploy runs {@link checkSlaves} too. The
 * deploy config schema checks that each channel has a device and an entry.
 * @param slaves - The slaves that the channels of the task use.
 * @param onRack - The slaves on the rack of the task.
 */
export const slavesZ = (
  pdoType: PDOType,
  slaves: SlaveDevice[],
  onRack: SlaveOnRack[],
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
  onRack: SlaveOnRack[],
): void => {
  const [problem] = problemsOf(pdoType, config, slaves, onRack);
  if (problem != null) throw new Error(problem.message);
};
