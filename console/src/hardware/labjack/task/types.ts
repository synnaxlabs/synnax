// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, task } from "@synnaxlabs/client";
import { z } from "zod";

// READS

export const readChan = z.object({
  port: z.number(),
  enabled: z.boolean(),
  dataType: z.string(),
  key: z.string(),
  range: z.number().optional(),
  channel: z.number(),
  type: z.literal("AIN").or(z.literal("DIN")),
});

export type ReadChan = z.infer<typeof readChan>;
export type ReadChanType = ReadChan["type"];

export const ZERO_READ_CHAN: ReadChan = {
  port: 0,
  enabled: true,
  dataType: "float32",
  key: "",
  channel: 0,
  type: "AIN",
  range: 0,
};

export const writeChan = z.object({
  port: z.number(),
  enabled: z.boolean(),
  dataType: z.string(),
  cmdKey: z.number(),
  stateKey: z.number(),
  key: z.string(),
  channelType: z.literal("AO").or(z.literal("DO")),
});

export type WriteChan = z.infer<typeof writeChan>;
export type WriteChanType = WriteChan["channelType"];
export const ZERO_WRITE_CHAN: WriteChan = {
  port: 0,
  enabled: true,
  key: "",
  dataType: "float32",
  cmdKey: 0,
  stateKey: 0,
  channelType: "DO",
};

const deviceKeyZ = device.deviceKeyZ.min(1, "Must specify a device");

export const readTaskConfigZ = z
  .object({
    sampleRate: z.number().int().min(0).max(50000),
    streamRate: z.number().int().min(0).max(50000),
    deviceKey: deviceKeyZ,
    channels: z.array(readChan),
    dataSaving: z.boolean(),
    indexKeys: z.array(z.number()),
  })
  .refine(
    (cfg) =>
      // Ensure that the stream Rate is lower than the sample rate
      cfg.sampleRate >= cfg.streamRate,
    {
      path: ["streamRate"],
      message: "Stream rate must be less than or equal to the sample rate",
    },
  );
export type ReadTaskConfig = z.infer<typeof readTaskConfigZ>;

export const writeTaskConfigZ = z.object({
  deviceKey: deviceKeyZ,
  channels: z.array(writeChan),
  dataSaving: z.boolean(),
  stateRate: z.number().int(),
});
export type WriteTaskConfig = z.infer<typeof writeTaskConfigZ>;

export const baseReadStateDetailsZ = z.object({
  running: z.boolean(),
  message: z.string(),
});
type baseReadStateDetails = z.infer<typeof baseReadStateDetailsZ>;

export const errorReadStateDetailsZ = baseReadStateDetailsZ.extend({
  errors: z.array(
    z.object({
      message: z.string(),
      path: z.string(),
    }),
  ),
});
type ErrorReadStateDetails = z.infer<typeof errorReadStateDetailsZ>;

export type ReadStateDetails = baseReadStateDetails | ErrorReadStateDetails;

export const writeStateDetailsZ = z.object({ running: z.boolean() });
export type WriteStateDetails = z.infer<typeof writeStateDetailsZ>;

export const READ_TYPE = "labjack_read";
export type ReadType = typeof READ_TYPE;

export const ZERO_READ_CONFIG: ReadTaskConfig = {
  deviceKey: "",
  sampleRate: 10,
  streamRate: 5,
  channels: [],
  indexKeys: [],
  dataSaving: true,
};
export type Read = task.Task<ReadTaskConfig, ReadStateDetails, ReadType>;
export type ReadPayload = task.Payload<ReadTaskConfig, ReadStateDetails, ReadType>;
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "LabJack Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export const WRITE_TYPE = "labjack_write";
export type WriteType = typeof WRITE_TYPE;

export const ZERO_WRITE_CONFIG: WriteTaskConfig = {
  deviceKey: "",
  channels: [],
  dataSaving: true,
  stateRate: 10,
};
export type Write = task.Task<WriteTaskConfig, WriteStateDetails, WriteType>;
export type WritePayload = task.Payload<WriteTaskConfig, WriteStateDetails, WriteType>;
export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "LabJack Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};

export type Chan = ReadChan | WriteChan;
