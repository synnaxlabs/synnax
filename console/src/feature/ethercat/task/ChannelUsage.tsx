// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ethercat/task/ChannelUsage.css";

import { type channel, query } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { Channel, Flux, License, Synnax } from "@synnaxlabs/pluto";
import { primitive, unique } from "@synnaxlabs/x";
import { type ReactElement, useEffect } from "react";

import { type SlaveDevice } from "@/feature/ethercat/device/types";
import {
  type Channel as TaskChannel,
  type DataTypes,
  hasEntry,
  type PDOType,
  readChannelOf,
  writeChannelOf,
} from "@/feature/ethercat/task/types";
import { CSS } from "@/platform/css";

const { useResult: useResultCount, useRefresh: useRefreshCount } = Flux.createRetrieve<
  Record<string, never>,
  number
>({
  name: "channel count",
  retrieve: async ({ client }) => await client.channels.count(),
});

// A fetch that discovers a channel fills the cache. It does not change the count.
const changesCount = (event: query.TableEvent<channel.Key, channel.Channel>): boolean =>
  event.variant === "delete" || event.discovered !== true;

/** @returns the count of the channels that the license counts, kept current. */
const useCount = (): number | undefined => {
  const client = Synnax.use();
  const refresh = useRefreshCount();
  useEffect(() => {
    if (client == null) return;
    // The cache keeps the count of an earlier mount, which missed the changes since.
    refresh({});
    return client.channels.store.subscribeBatch((events) => {
      if (events.some(changesCount)) refresh({});
    });
  }, [client, refresh]);
  return useResultCount({}).data;
};

/** @returns the keys of the channels that the slaves map, indexes among them. */
const mappedKeys = (slaves: SlaveDevice[]): number[] =>
  unique.unique(
    slaves
      .flatMap(({ properties: p }) => [
        p.readIndex,
        p.writeStateIndex,
        ...p.read.channels.map(({ channel }) => channel),
        ...p.write.channels.flatMap(({ command, state }) => [command, state]),
      ])
      .filter(primitive.isNonZero),
  );

/**
 * @returns how many channels a Start creates on the slaves: an index for each slave
 * whose index is gone, and each channel that an enabled channel needs and the slave
 * does not map. existing holds the data types of the mapped channels that exist.
 */
const createsOf = (
  pdoType: PDOType,
  slaves: SlaveDevice[],
  channels: TaskChannel[],
  existing: DataTypes,
): number =>
  slaves.reduce((count, slave) => {
    const missing = (key: number) => (existing.has(key) ? 0 : 1);
    let needed = 0;
    for (const ch of channels) {
      if (ch.device !== slave.key || ch.disabled || !hasEntry(ch)) continue;
      if ("calibrated" in ch) needed += missing(readChannelOf(slave, ch, existing));
      else
        needed +=
          missing(writeChannelOf(slave, ch, "command")) +
          missing(writeChannelOf(slave, ch, "state"));
    }
    const { readIndex, writeStateIndex } = slave.properties;
    return count + needed + missing(pdoType === "inputs" ? readIndex : writeStateIndex);
  }, 0);

const channelsText = (count: number): string => {
  if (count === 0) return "no channels";
  return count === 1 ? "1 channel" : `${count} channels`;
};

export interface ChannelUsageProps {
  pdoType: PDOType;
  channels: TaskChannel[];
  /** The slaves that the channels use, or undefined while they load. */
  slaves: SlaveDevice[] | undefined;
}

/**
 * Says how many channels a Start creates and how many of the channels that the license
 * allows are in use. Renders nothing until both counts load.
 */
export const ChannelUsage = ({
  pdoType,
  channels,
  slaves,
}: ChannelUsageProps): ReactElement | null => {
  const keys = mappedKeys(slaves ?? []);
  const { data: mapped } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  const inUse = useCount();
  const { data: info } = License.useResult({});
  if (slaves == null || inUse == null) return null;
  if (keys.length > 0 && mapped == null) return null;
  const existing: DataTypes = new Map(
    (mapped ?? []).map((c) => [c.key, c.dataType.toString()]),
  );
  const creates = createsOf(pdoType, slaves, channels, existing);
  const cap = info?.license?.channels ?? 0;
  let message = `Start creates ${channelsText(creates)}, ${inUse}`;
  message += cap === 0 ? " in use." : ` of ${cap} in use.`;
  const over = cap === 0 ? 0 : inUse + creates - cap;
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
