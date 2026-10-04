// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, mqtt } from "@synnaxlabs/client";

import { type SparkplugDataType, type SparkplugTagID } from "@/feature/mqtt/task/types";

const NAMESPACE = "spBv1.0";

const levels = ({ group, edgeNode, device, tag }: SparkplugTagID): string[] => [
  group,
  edgeNode,
  ...(device === "" ? [] : [device]),
  tag,
];

/**
 * @returns the key that holds the channels of a tag in the broker properties. The
 * device level is always present, because a tag name can hold a slash.
 */
export const sparkplugPropertiesKey = ({
  group,
  edgeNode,
  device,
  tag,
}: SparkplugTagID): string => [NAMESPACE, group, edgeNode, device, tag].join("/");

/** @returns the name prefix of the channels that a configure creates for a tag. */
export const sparkplugChannelName = (deviceName: string, id: SparkplugTagID): string =>
  channel.escapeInvalidName([deviceName, ...levels(id)].join("_"));

const DATA_TYPES: Record<SparkplugDataType, string> = {
  int8: "int8",
  int16: "int16",
  int32: "int32",
  int64: "int64",
  uint8: "uint8",
  uint16: "uint16",
  uint32: "uint32",
  uint64: "uint64",
  float: "float32",
  double: "float64",
  boolean: "uint8",
  string: "string",
  date_time: "timestamp",
};

/** @returns the Synnax data type that holds values of a Sparkplug B data type. */
export const fromSparkplugDataType = (type: SparkplugDataType): string =>
  DATA_TYPES[type];

const STRING_DATA_TYPES = ["Text", "UUID"];

/**
 * Parses the data type of a browsed tag. Text and UUID tags carry strings.
 * @throws {z.ZodError} if a task cannot take a tag of the data type.
 */
export const toSparkplugDataType = (browsed: string): SparkplugDataType =>
  mqtt.sparkplugDataTypeZ.parse(
    STRING_DATA_TYPES.includes(browsed) ? "string" : browsed,
  );
