// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, mqtt } from "@synnaxlabs/client";
import { json } from "@synnaxlabs/x";

import type { Device } from "@/feature/mqtt/device";
import { createLeaves, createRawLeaf, type Leaf } from "@/feature/mqtt/task/leaves";
import {
  fromSparkplugDataType,
  toSparkplugDataType,
} from "@/feature/mqtt/task/sparkplug";
import type { SparkplugReadEntry, SparkplugTag } from "@/feature/mqtt/task/types";

/** The prefix configure gives the channels of a topic's fields. */
export const namePrefix = (devName: string, topic: string): string =>
  `${channel.escapeInvalidName(devName)}_${channel.escapeInvalidName(topic)}`;

export const createSparkplugEntry = (tag?: SparkplugTag): SparkplugReadEntry => {
  const entry = mqtt.sparkplugReadEntryZ.parse({ type: "sparkplug" });
  if (tag == null) return entry;
  const { dataType, ...tagID } = tag;
  return {
    ...entry,
    ...tagID,
    dataType: fromSparkplugDataType(toSparkplugDataType(dataType)),
  };
};

/** Names a pointer for people: "motor › speed", or "Whole payload". */
export const pointerLabel = (pointer: string): string =>
  pointer === "" ? "Whole payload" : json.parsePointer(pointer).join(" › ");

/** The values that the messages of one shape of a browsed topic offer. */
export interface Shape {
  /** The sorted pointers of the shape, which stay the same across browses. */
  key: string;
  leaves: Leaf[];
  /** The last value heard at each pointer. */
  samples: Map<string, string>;
  /** The pointer of the timestamp that indexes the messages. */
  index?: string;
  /** The pointer of the field that only the messages of the shape hold. */
  present?: string;
  /** The number of messages heard. Absent for a payload that is not JSON. */
  count?: number;
}

/**
 * @returns the shapes of the messages of a browsed topic. A topic with no JSON message
 * offers its last payload as one string.
 */
export const shapesOf = ({ payload, shapes }: Device.BrowsedTopic): Shape[] => {
  if (shapes.length === 0)
    return payload === ""
      ? []
      : [{ key: "", leaves: [createRawLeaf()], samples: new Map([["", payload]]) }];
  return shapes.map(({ fields, index, present, count }) => ({
    key: fields
      .map(({ pointer }) => pointer)
      .sort()
      .join("\n"),
    leaves: createLeaves(fields),
    samples: new Map(fields.map(({ pointer, sample }) => [pointer, sample])),
    index,
    present,
    count,
  }));
};
