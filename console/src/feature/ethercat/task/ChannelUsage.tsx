// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ethercat/task/ChannelUsage.css";

import { channel, query } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, License } from "@synnaxlabs/pluto";
import { primitive, unique } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { type IndexOptions } from "@/feature/ethercat/task/configure";
import {
  type Channel as TaskChannel,
  type DataTypes,
  entryAt,
  hasEntry,
  mappedChannels,
  type PDOType,
  readChannelName,
  readChannelOf,
  writeChannelNames,
  writeChannelOf,
} from "@/feature/ethercat/task/types";
import { CSS } from "@/platform/css";

/** @returns the keys of the channels that the slaves map, indexes among them. */
const mappedKeys = (slaves: SlaveDevice[]): number[] =>
  unique.unique(
    slaves
      .flatMap((s) => [
        s.properties.readIndex,
        s.properties.writeStateIndex,
        ...mappedChannels(s),
      ])
      .filter(primitive.isNonZero),
  );

/** A channel that a Start creates when no channel holds its name. */
interface Creation {
  name: string;
  /** How many channels the Start creates for it. */
  count: number;
}

/**
 * @param existing - The data types of the mapped channels that exist.
 * @returns the channels a Start creates before it reuses names: the index of each slave
 * whose index is gone, and each channel an enabled channel needs that the slave does
 * not map.
 */
const creationsOf = (
  { indexProperty, nameSuffix }: IndexOptions,
  pdoType: PDOType,
  slaves: SlaveDevice[],
  channels: TaskChannel[],
  existing: DataTypes,
): Creation[] =>
  slaves.flatMap((slave) => {
    const { identifier, pdos } = slave.properties;
    // A new index clears the channels that the slave maps.
    const indexGone = !existing.has(slave.properties[indexProperty]);
    const gone = (key: number) => indexGone || !existing.has(key);
    const creations: Creation[] = [];
    if (indexGone)
      creations.push({
        name: `${channel.escapeInvalidName(identifier)}${nameSuffix}`,
        count: 1,
      });
    for (const ch of channels) {
      if (ch.device !== slave.key || ch.disabled || !hasEntry(ch)) continue;
      if (entryAt(pdos[pdoType], ch) == null) continue;
      if ("calibrated" in ch) {
        if (gone(readChannelOf(slave, ch, existing)))
          creations.push({ name: readChannelName(slave, ch, existing), count: 1 });
        continue;
      }
      const { cmd, state } = writeChannelNames(slave, ch);
      // A new command channel gets an index of its own.
      if (gone(writeChannelOf(slave, ch, "command")))
        creations.push({ name: cmd, count: 2 });
      if (gone(writeChannelOf(slave, ch, "state")))
        creations.push({ name: state, count: 1 });
    }
    return creations;
  });

const channelsText = (count: number): string => {
  if (count === 0) return "no channels";
  return count === 1 ? "1 channel" : `${count} channels`;
};

export interface ChannelUsageProps {
  indexOptions: IndexOptions;
  pdoType: PDOType;
  channels: TaskChannel[];
  /** The slaves that the channels use, or undefined while they load. */
  slaves: SlaveDevice[] | undefined;
}

/**
 * Says how many channels a Start creates and how many of the channels that the license
 * allows are in use. Renders nothing while the counts load.
 */
export const ChannelUsage = ({
  indexOptions,
  pdoType,
  channels,
  slaves,
}: ChannelUsageProps): ReactElement | null => {
  const keys = mappedKeys(slaves ?? []);
  const { data: mapped } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  const existing: DataTypes | undefined =
    keys.length > 0 && mapped == null
      ? undefined
      : new Map((mapped ?? []).map((c) => [c.key, c.dataType.toString()]));
  const creations =
    slaves == null || existing == null
      ? undefined
      : creationsOf(indexOptions, pdoType, slaves, channels, existing);
  const names = unique.unique((creations ?? []).map(({ name }) => name));
  const { data: named } = Channel.useResultMultiple(
    names.length === 0 ? null : { where: { name: { in: names } } },
  );
  const count = Channel.useResultCountLimited();
  const license = License.useResult({});
  if (creations == null || (names.length > 0 && named == null)) return null;
  if (license.variant === "loading") return null;
  const inUse = count.data;
  // A count that fails, as for a user who cannot retrieve channels, leaves it out.
  if (inUse == null && count.variant !== "error") return null;
  const held = new Set((named ?? []).map(({ name }) => name));
  const creates = creations.reduce((n, c) => (held.has(c.name) ? n : n + c.count), 0);
  const cap = license.data?.license?.channels ?? 0;
  let message = `Start creates ${channelsText(creates)}`;
  if (inUse == null) message += ".";
  else message += cap === 0 ? `, ${inUse} in use.` : `, ${inUse} of ${cap} in use.`;
  const over = cap === 0 || inUse == null ? 0 : inUse + creates - cap;
  if (over > 0) {
    message += ` Delete ${channelsText(over)} that you do not use`;
    message += " before you start the task.";
  }
  return (
    <Text.Text
      level="small"
      color={over > 0 ? undefined : 9}
      status={over > 0 ? "warning" : undefined}
      className={CSS.BE("ethercat-channel-usage", "text")}
    >
      {message}
    </Text.Text>
  );
};
