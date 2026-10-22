// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, mqtt } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";

import { type SparkplugHaulTag } from "@/feature/mqtt/device/SparkplugBrowser";
import { createReadFields } from "@/feature/mqtt/task/createReadFields";
import {
  fromSparkplugDataType,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import {
  type PlainReadEntry,
  type ReadField,
  type SparkplugReadEntry,
} from "@/feature/mqtt/task/types";

export const ENTRIES_PATH = "config.entries";

/** The prefix configure gives the channels of a topic's fields. */
export const namePrefix = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}`;

export const createEntry = (): PlainReadEntry =>
  mqtt.plainReadEntryZ.parse({ type: "plain" });

export const createSparkplugEntry = (tag?: SparkplugHaulTag): SparkplugReadEntry => {
  const entry = mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug" });
  if (tag == null) return entry;
  const { dataType, ...tagID } = tag;
  return {
    ...entry,
    ...tagID,
    dataType: fromSparkplugDataType(toSparkplugDataType(dataType)),
  };
};

const unescapeToken = (token: string): string =>
  token.replaceAll("~1", "/").replaceAll("~0", "~");

const tokens = (pointer: string): string[] =>
  pointer === "" ? [] : pointer.slice(1).split("/").map(unescapeToken);

/** Names a pointer for people: "motor › speed", or "Whole payload". */
export const pointerLabel = (pointer: string): string =>
  pointer === "" ? "Whole payload" : tokens(pointer).join(" › ");

const format = (value: unknown): string =>
  typeof value === "string" ? `"${value}"` : JSON.stringify(value);

/** @returns the value at a pointer in a payload sample, or null when it has none. */
export const sampleAt = (payload: string, pointer: string): string | null => {
  let value: unknown;
  try {
    value = JSON.parse(payload);
  } catch {
    return pointer === "" ? payload : null;
  }
  for (const token of tokens(pointer)) {
    if (value == null || typeof value !== "object") return null;
    value = (value as Record<string, unknown>)[token];
  }
  return value === undefined ? null : format(value);
};

/**
 * The fields a payload sample offers. A payload that is not JSON offers itself as one
 * string field.
 */
export const leavesOf = (payload: string): ReadField[] => {
  const fields = createReadFields(payload);
  if (fields.length > 0 || payload === "") return fields;
  return [
    { ...mqtt.readFieldZ.parse({}), pointer: "", dataType: DataType.STRING.toString() },
  ];
};
