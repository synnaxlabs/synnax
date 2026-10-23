// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type icd } from "@synnaxlabs/client";

import { fieldOf } from "@/feature/bus/types";

const safe = (name: string): string => channel.escapeInvalidName(name, true);

interface Selector {
  multiplexor: icd.BaseField;
  values: number[];
}

/** @returns the multiplexors that select a field, outermost first. */
const selectorsOf = (entry: icd.MessageEntry, field: icd.BaseField): Selector[] => {
  if (field.multiplexor == null) return [];
  const multiplexor = fieldOf(entry, field.multiplexor);
  const values = [...new Set(field.multiplexValues)].sort((a, b) => a - b);
  return [...selectorsOf(entry, multiplexor), { multiplexor, values }];
};

/**
 * @returns the name of the index a read task stamps a field's samples on. Fields that
 * the same multiplexor values select share an index. With one multiplexor in the
 * message, the name holds the values (mux1_2). With more, it holds each multiplexor and
 * its values (Mode_3_Sub_5). An always-present field uses the plain index.
 * @throws {Error} if the message has no such field.
 */
export const indexName = (
  device: string,
  entry: icd.MessageEntry,
  field: icd.FieldKey,
): string => {
  const base = `${safe(device)}_${safe(entry.name)}`;
  const selectors = selectorsOf(entry, fieldOf(entry, field));
  if (selectors.length === 0) return `${base}_time`;
  const multiplexors = new Set(
    entry.payload.fields.flatMap((f) => f.multiplexor ?? []),
  );
  const group =
    multiplexors.size === 1
      ? `mux${selectors[0].values.join("_")}`
      : selectors.map((s) => [s.multiplexor.name, ...s.values].join("_")).join("_");
  return `${base}_${safe(group)}_time`;
};

/** @returns the name of the channel a read task writes a field to. */
export const fieldChannelName = (device: string, message: string, field: string) =>
  `${safe(device)}_${safe(message)}_${safe(field)}`;

/** @returns the name of the index channel of a message's command channels. */
export const commandIndexName = (device: string, message: string): string =>
  `${safe(device)}_${safe(message)}_cmd_time`;

/** @returns the name of the command channel a write task encodes a field from. */
export const commandChannelName = (device: string, message: string, field: string) =>
  `${fieldChannelName(device, message, field)}_cmd`;

/** @returns the name of the virtual channel a read task writes raw frames to. */
export const rawName = (device: string): string => `${safe(device)}_raw`;
