// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { json, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { device } from "@/device";
import { driver } from "@/driver";

/** A topic of a cluster, as a browse command of the Kafka scan task sees it. */
export const topicZ = z.object({
  name: z.string(),
  partitions: z.number(),
  /** records is the number of records that the partitions hold. */
  records: z.number(),
  /** compacted is true when the cluster keeps only the newest record of each key. */
  compacted: z.boolean(),
  /** rate is the number of records per second over the last minute. */
  rate: z.number(),
  /** newest is the time of the newest record. Zero when the topic holds none. */
  newest: TimeStamp.z,
  /** format is the format of the newest record. Undefined when the topic holds none. */
  format: driver.formatZ.optional(),
  /** readable is false when the cluster does not let the Core read the topic. */
  readable: z.boolean(),
});

export interface Topic extends z.infer<typeof topicZ> {}

/** The data that answers a browse command of the Kafka scan task. */
export const browseResultZ = z.object({
  /** topics are the topics of the cluster, sorted by name. */
  topics: z.array(topicZ),
});

export interface BrowseResult extends z.infer<typeof browseResultZ> {}

/** The arguments of a sample command of the Kafka scan task. */
export const sampleArgsZ = z.object({
  /** device is the key of the cluster device. */
  device: device.keyZ,
  /** topics are the topics to read. */
  topics: z.array(z.string()),
  /** count is the most records to read from each topic. Defaults to 100. */
  count: z.number().optional(),
});

export interface SampleArgs extends z.input<typeof sampleArgsZ> {}

export const headerZ = z.object({ key: z.string(), value: z.string() });

export interface Header extends z.infer<typeof headerZ> {}

/** A record that a sample read from a topic. */
export const sampledRecordZ = z.object({
  partition: z.number(),
  offset: z.number(),
  key: z.string(),
  time: TimeStamp.z,
  headers: z.array(headerZ),
  /** payload holds at most the first 16 KiB of the record value. */
  payload: driver.payloadZ,
});

export interface SampledRecord extends z.infer<typeof sampledRecordZ> {}

/** The records that a sample read from one topic, and the shapes of their values. */
export const sampledTopicZ = z.object({
  topic: z.string(),
  records: z.array(sampledRecordZ),
  /** shapes group the records read by the set of fields that their values hold. */
  shapes: z.array(json.shapeZ),
  /** error says why the topic could not be read. Empty when it was read. */
  error: z.string(),
});

export interface SampledTopic extends z.infer<typeof sampledTopicZ> {}

/** The data that answers a sample command of the Kafka scan task. */
export const sampleResultZ = z.object({
  /** topics are the samples, in the order of the topics of the arguments. */
  topics: z.array(sampledTopicZ),
  /**
   * truncated is true when the records read do not all fit in the answer. The shapes
   * still come from every record read.
   */
  truncated: z.boolean(),
});

export interface SampleResult extends z.infer<typeof sampleResultZ> {}
