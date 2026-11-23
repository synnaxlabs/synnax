// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, query } from "@synnaxlabs/client";
import { Channel } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";

import type { Task } from "@/platform/task";

export interface DataTypes {
  /** The data type of each stored channel that exists, by key. */
  dataTypes: ReadonlyMap<channel.Key, string>;
  /**
   * True while the data types load. Each control that binds waits for them, as the
   * Core refuses a channel that a field does not convert to.
   */
  bindDisabled: boolean;
}

/** @returns the data types of the channels in read, the read properties of a device. */
export const useDataTypes = (
  read: Record<string, Pick<Task.ReadChannelProps, "channels">>,
): DataTypes => {
  const keys = unique
    .unique(Object.values(read).flatMap(({ channels }) => Object.values(channels)))
    .filter((key) => key !== 0);
  const { data, variant } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  return {
    dataTypes: new Map(data?.map(({ key, dataType }) => [key, dataType.toString()])),
    bindDisabled: keys.length > 0 && variant !== "success",
  };
};
