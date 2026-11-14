// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ethercat } from "@synnaxlabs/client";
import { type Flux } from "@synnaxlabs/pluto";
import { DataType, primitive, unique } from "@synnaxlabs/x";
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
 * @returns why the slave cannot start because another device on the rack is the same
 * module: the same vendor, product, and serial number. The problem names the device
 * that is not connected, which the user deletes. A slave with no serial number has no
 * problem, as the Driver cannot tell its module apart.
 */
const serialProblem = (slave: SlaveDevice, onRack: SlaveDevice[]): string | null => {
  const { vendorId, productCode, serial } = slave.properties;
  if (serial === 0) return null;
  const other = onRack.find(
    ({ key, properties: p }) =>
      key !== slave.key &&
      p.vendorId === vendorId &&
      p.productCode === productCode &&
      p.serial === serial,
  );
  if (other == null) return null;
  const connected = (d: SlaveDevice): boolean =>
    onRack.find((o) => o.key === d.key)?.status?.details.connected === true;
  const stale = connected(slave) ? other : connected(other) ? slave : null;
  const step =
    stale == null
      ? "Delete the one that is not on the network."
      : `Delete ${stale.name}, which is not connected.`;
  const pair = `Slaves ${slave.name} and ${other.name}`;
  return `${pair} have the same serial number ${serial}. ${step}`;
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

/**
 * @returns why the check cannot run yet on the slaves of the query, or null. A query
 * that loads with earlier data that holds every key is ready.
 */
const loadProblem = (
  query: Flux.Result<SlaveDevice[]>,
  keys: device.Key[],
): string | null => {
  if (query.variant === "error") {
    const { message, description } = query.status;
    return description == null ? `${message}.` : `${message}. ${description}`;
  }
  if (query.variant === "success") return null;
  const loaded = query.data ?? [];
  if (keys.every((k) => loaded.some((s) => s.key === k))) return null;
  return "Loading the slaves of the task.";
};

/** The parts of the config of an EtherCAT task that its checks read. */
interface CheckedConfig {
  channels: Channel[];
  /** The settings that a read task writes. A write task has none. */
  startupParameters?: ethercat.SlaveStartupParameters[];
}

/**
 * Checks the channels of a task against their slaves: one network for every slave, no
 * other device with the serial number of a slave, and for each enabled channel its
 * entry and, on inputs, the settings of the input it reads. Until both queries hold
 * the slaves they need, the check reports that they load, so a deploy cannot skip it.
 * @param query - The query of the slaves that the channels of the task use.
 * @param onRack - The query of the slaves on the rack of the task, with statuses.
 */
export const slavesZ = (
  pdoType: PDOType,
  query: Flux.Result<SlaveDevice[]>,
  onRack: Flux.Result<SlaveDevice[]>,
): z.ZodType =>
  Task.check<CheckedConfig>(({ channels, startupParameters = [] }, report) => {
    const keys = unique.unique(
      channels.map(({ device }) => device).filter(primitive.isNonZero),
    );
    const slaves = query.data ?? [];
    const used = slaves.filter((s) => keys.includes(s.key));
    const usedKeys = used.map(({ key }) => key);
    const unloaded = loadProblem(query, keys) ?? loadProblem(onRack, usedKeys);
    if (unloaded != null) return report(["channels"], unloaded);
    const onRackSlaves = onRack.data ?? [];
    const problems = [
      networkProblem(used),
      ...used.map((slave) => serialProblem(slave, onRackSlaves)),
    ].filter((p): p is string => p != null);
    if (problems.length > 0) report(["channels"], problems.join(" "));
    channels.forEach((ch, i) => {
      const slave = used.find((s) => s.key === ch.device);
      if (slave == null || ch.disabled) return;
      const enabled = channels.filter((o) => !o.disabled && o.device === ch.device);
      let problem = entryProblem(slave, enabled, ch, pdoType);
      if (problem == null && pdoType === "inputs" && hasEntry(ch))
        problem = settingProblem(slave, ch, startupParameters);
      if (problem != null) report(["channels", i], problem);
    });
  });
