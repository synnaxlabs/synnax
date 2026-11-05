// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { json, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { device } from "@/device";
import { driver } from "@/driver";

const TEST_ENDPOINT = "/mqtt/test";
const BROWSE_ENDPOINT = "/mqtt/browse";
const SAMPLE_ENDPOINT = "/mqtt/sample";

const millisecondsZ = TimeSpan.z.transform((span) => span.milliseconds);

export const testRequestZ = z.object({
  /**
   * device is the key of the broker device that the test edits. Undefined for a broker
   * that is not a device yet.
   */
  device: device.keyZ.optional(),
  /** location is the host of the broker. */
  location: z.string(),
  /** properties are the connection properties of the broker device to test. */
  properties: z.record(z.string(), z.unknown()),
});

export interface TestRequest extends z.input<typeof testRequestZ> {}

const testResZ = z.object({ steps: z.array(driver.testStepZ) });

export const browseRequestZ = z.object({
  /** device is the key of the broker device. */
  device: device.keyZ,
  /** filter selects the topics to listen to. Defaults to every topic. */
  filter: z.string().optional(),
  /** duration is how long to listen. Defaults to 5 seconds. */
  duration: millisecondsZ.optional(),
  /** limit is the most topics to list. */
  limit: z.number().optional(),
});

export interface BrowseRequest extends z.input<typeof browseRequestZ> {}

/** A topic that the broker delivered while the Core listened. */
export const browsedTopicZ = z.object({
  topic: z.string(),
  /** count is the number of messages that arrived. */
  count: z.number(),
  /** rate is the number of messages per second. */
  rate: z.number(),
  lastSeen: TimeStamp.z,
  /** payload holds at most the first 256 bytes of the last message. */
  payload: driver.payloadZ,
  format: driver.formatZ,
  /** retained is true when the last message was a retained message. */
  retained: z.boolean(),
});

export interface BrowsedTopic extends z.infer<typeof browsedTopicZ> {}

export const browseResultZ = z.object({
  /** topics are the topics that delivered messages, sorted by name. */
  topics: z.array(browsedTopicZ),
  /** truncated is true when more topics delivered messages than the limit. */
  truncated: z.boolean(),
});

export interface BrowseResult extends z.infer<typeof browseResultZ> {}

export const sampleRequestZ = z.object({
  /** device is the key of the broker device. */
  device: device.keyZ,
  /** topics are the topics or topic filters to read. */
  topics: z.array(z.string()),
  /** count is the most messages to keep from each topic. Defaults to 20. */
  count: z.number().optional(),
  /** duration is the longest time to listen. Defaults to 5 seconds. */
  duration: millisecondsZ.optional(),
});

export interface SampleRequest extends z.input<typeof sampleRequestZ> {}

export const sampledMessageZ = z.object({
  time: TimeStamp.z,
  /** payload holds at most the first 16 KiB of the message. */
  payload: driver.payloadZ,
  retained: z.boolean(),
});

export interface SampledMessage extends z.infer<typeof sampledMessageZ> {}

export const sampledTopicZ = z.object({
  topic: z.string(),
  messages: z.array(sampledMessageZ),
});

export interface SampledTopic extends z.infer<typeof sampledTopicZ> {}

export const sampleResultZ = z.object({
  topics: z.array(sampledTopicZ),
  /** shapes group the topics by the set of fields that their payloads hold. */
  shapes: z.array(json.shapeZ),
  /** truncated is true when the broker delivered more than a sample keeps. */
  truncated: z.boolean(),
});

export interface SampleResult extends z.infer<typeof sampleResultZ> {}

/** Tests connections to MQTT brokers and reads the topics of broker devices. */
export class Client {
  private readonly unary: UnaryClient;

  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Tests a connection to a broker, one stage at a time.
   * @returns the stages that ran, in order. The last is the first that failed.
   * @throws {AccessDeniedError} if the caller cannot update the device, or cannot
   * create devices when the request names no device.
   */
  async test(request: TestRequest): Promise<driver.TestStep[]> {
    const { steps } = await this.unary.send(
      TEST_ENDPOINT,
      request,
      testRequestZ,
      testResZ,
    );
    return steps;
  }

  /**
   * Listens to a broker device for a duration and lists the topics that delivered
   * messages.
   * @throws {AccessDeniedError} if the caller cannot retrieve the device.
   */
  async browse(request: BrowseRequest): Promise<BrowseResult> {
    return await this.unary.send(
      BROWSE_ENDPOINT,
      request,
      browseRequestZ,
      browseResultZ,
    );
  }

  /**
   * Reads messages from topics of a broker device and infers the shapes of their
   * payloads.
   * @throws {AccessDeniedError} if the caller cannot retrieve the device.
   */
  async sample(request: SampleRequest): Promise<SampleResult> {
    return await this.unary.send(
      SAMPLE_ENDPOINT,
      request,
      sampleRequestZ,
      sampleResultZ,
    );
  }
}
