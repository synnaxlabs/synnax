// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, icd, query } from "@synnaxlabs/client";
import { Channel } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";

import { converts } from "@/platform/icd/dataType";
import type { Task } from "@/platform/task";

/**
 * @returns the bindings of fields, each to the channel stored for its pointer. A field
 * gets no channel when its stored channel is in written, as the Core refuses a channel
 * that two bindings write, or when its kind does not convert to the data type of the
 * channel in dataTypes. Adds the channels it binds to written.
 */
export const bindingsOf = (
  stored: Record<string, channel.Key>,
  fields: Pick<icd.JSONField, "key" | "pointer" | "type">[],
  written: Set<channel.Key>,
  dataTypes: ReadonlyMap<channel.Key, string>,
): icd.ReadBinding[] =>
  fields.map(({ key: field, pointer, type }) => {
    const ch = stored[pointer] ?? 0;
    const dataType = dataTypes.get(ch);
    if (written.has(ch) || dataType == null || !converts(type, dataType))
      return icd.readBindingZ.parse({ field });
    written.add(ch);
    return icd.readBindingZ.parse({ field, channel: ch });
  });

export interface StoredDataTypes {
  /** The data type of each stored channel that exists, by key. */
  dataTypes: ReadonlyMap<channel.Key, string>;
  /**
   * True while the data types load. Each control that binds waits for them, as the
   * Core refuses a channel that the kind of a field does not convert to.
   */
  bindDisabled: boolean;
}

/** @returns the data types of the channels in read, the read properties of a device. */
export const useStoredDataTypes = (
  read: Record<string, Pick<Task.ReadChannelProps, "channels">>,
): StoredDataTypes => {
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
