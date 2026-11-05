// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { json, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { device } from "@/device";
import { driver } from "@/driver";
import { propertiesZ } from "@/kafka/types.gen";

const TEST_ENDPOINT = "/kafka/test";
const BROWSE_ENDPOINT = "/kafka/browse";
const SAMPLE_ENDPOINT = "/kafka/sample";

/** A topic of a cluster. */
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
  /** shapes group the records by the set of fields that their values hold. */
  shapes: z.array(json.shapeZ),
  /** error says why the topic could not be read. Empty when it was read. */
  error: z.string(),
});

export interface SampledTopic extends z.infer<typeof sampledTopicZ> {}

const testReqZ = z.object({ device: device.keyZ.optional(), properties: propertiesZ });
const testResZ = z.object({ steps: z.array(driver.testStepZ) });
const browseReqZ = z.object({ device: device.keyZ });
const browseResZ = z.object({ topics: z.array(topicZ) });

export const sampleRequestZ = z.object({
  /** device is the key of the cluster device. */
  device: device.keyZ,
  /** topics are the topics to read. */
  topics: z.array(z.string()),
  /** count is the most records to read from each topic. Defaults to 100. */
  count: z.number().optional(),
});

export interface SampleRequest extends z.input<typeof sampleRequestZ> {}

const sampleResZ = z.object({ topics: z.array(sampledTopicZ) });

/** Tests connections to Kafka clusters and reads the topics of cluster devices. */
export class Client {
  private readonly unary: UnaryClient;

  constructor(unary: UnaryClient) {
    this.unary = unary;
  }

  /**
   * Tests a connection to a cluster, one stage at a time.
   * @param properties - the connection properties to test.
   * @param device - the key of the cluster device that the properties edit. Undefined
   * for a cluster that is not a device yet.
   * @returns the stages that ran, in order. The last is the first that failed.
   * @throws {AccessDeniedError} if the caller cannot update the device, or cannot
   * create devices when device is undefined.
   */
  async test(
    properties: z.input<typeof propertiesZ>,
    device?: device.Key,
  ): Promise<driver.TestStep[]> {
    const { steps } = await this.unary.send(
      TEST_ENDPOINT,
      { device, properties },
      testReqZ,
      testResZ,
    );
    return steps;
  }

  /**
   * Lists the topics of a cluster device, sorted by name.
   * @throws {AccessDeniedError} if the caller cannot retrieve the device.
   */
  async browse(device: device.Key): Promise<Topic[]> {
    const { topics } = await this.unary.send(
      BROWSE_ENDPOINT,
      { device },
      browseReqZ,
      browseResZ,
    );
    return topics;
  }

  /**
   * Reads records from topics of a cluster device, from windows spread back in time,
   * and infers the shapes of their values.
   * @returns the sample of each topic, in the order of the request.
   * @throws {AccessDeniedError} if the caller cannot retrieve the device.
   */
  async sample(request: SampleRequest): Promise<SampledTopic[]> {
    const { topics } = await this.unary.send(
      SAMPLE_ENDPOINT,
      request,
      sampleRequestZ,
      sampleResZ,
    );
    return topics;
  }
}
