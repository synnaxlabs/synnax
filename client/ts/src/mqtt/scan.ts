// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { json, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { device } from "@/device";

/** The encodings of a message payload. */
export const FORMATS = ["json", "text", "binary"] as const;

export const formatZ = z.enum(FORMATS);

export type Format = z.infer<typeof formatZ>;

const millisecondsZ = TimeSpan.z.transform((span) => span.milliseconds);

/** The arguments of a browse command of the MQTT scan task. */
export const browseArgsZ = z.object({
  /** device is the key of the broker device. */
  device: device.keyZ,
  /** filter selects the topics to listen to. Defaults to every topic. */
  filter: z.string().optional(),
  /** duration is how long to listen. Defaults to 3 seconds, at most 20. */
  duration: millisecondsZ.optional(),
  /** limit is the most topics to list. Defaults to 500. */
  limit: z.number().optional(),
});

export interface BrowseArgs extends z.input<typeof browseArgsZ> {}

/** A topic that the broker delivered while the Core listened. */
export const browsedTopicZ = z.object({
  topic: z.string(),
  /** count is the number of messages that arrived. */
  count: z.number(),
  /** rate is the number of messages per second, without a retained message. */
  rate: z.number(),
  lastSeen: TimeStamp.z,
  /** payload is the start of the last payload. Empty when it is not text. */
  payload: z.string(),
  /** format is the format of the whole last payload. */
  format: formatZ,
  /** retained is true when the last message was a retained message. */
  retained: z.boolean(),
});

export interface BrowsedTopic extends z.infer<typeof browsedTopicZ> {}

/** The data that answers a browse command of the MQTT scan task. */
export const browseResultZ = z.object({
  /** topics are the topics that delivered messages, sorted by name. */
  topics: z.array(browsedTopicZ),
  /**
   * truncated is true when more topics delivered messages than the limit, or when the
   * topics do not all fit in the answer.
   */
  truncated: z.boolean(),
});

export interface BrowseResult extends z.infer<typeof browseResultZ> {}

/** The arguments of a sample command of the MQTT scan task. */
export const sampleArgsZ = z.object({
  /** device is the key of the broker device. */
  device: device.keyZ,
  /** topics are the topics or topic filters to read. */
  topics: z.array(z.string()),
  /** count is the most messages to keep from each topic. Defaults to 20. */
  count: z.number().optional(),
  /** duration is the longest time to listen. Defaults to 5 seconds, at most 20. */
  duration: millisecondsZ.optional(),
});

export interface SampleArgs extends z.input<typeof sampleArgsZ> {}

export const sampledMessageZ = z.object({
  time: TimeStamp.z,
  /** format is the format of the whole payload. */
  format: formatZ,
  /**
   * payload is the start of the payload, from at most 16 KiB: text when format is json
   * or text, and hex digits otherwise.
   */
  payload: z.string(),
  retained: z.boolean(),
});

export interface SampledMessage extends z.infer<typeof sampledMessageZ> {}

export const sampledTopicZ = z.object({
  topic: z.string(),
  messages: z.array(sampledMessageZ),
});

export interface SampledTopic extends z.infer<typeof sampledTopicZ> {}

/** The data that answers a sample command of the MQTT scan task. */
export const sampleResultZ = z.object({
  topics: z.array(sampledTopicZ),
  /** shapes group the topics by the set of fields that their payloads hold. */
  shapes: z.array(json.shapeZ),
  /**
   * truncated is true when the broker delivered more than a sample holds, or when the
   * messages read do not all fit in the answer. The shapes still come from every
   * message read.
   */
  truncated: z.boolean(),
});

export interface SampleResult extends z.infer<typeof sampleResultZ> {}
