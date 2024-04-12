// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { z } from "zod";

const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const deviceChannelProperties = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.string(),
});

type DeviceChannelProperties = z.infer<typeof deviceChannelProperties>;

export const devicePropertiesZ = z.object({
  connection: connectionConfigZ,
  channels: deviceChannelProperties.array(),
});

export type DeviceProperties = z.infer<typeof devicePropertiesZ>;

export type ReadTaskChannelConfig = z.infer<typeof readTaskChannelConfigZ>;

export const readTaskChannelConfigZ = z.object({
  key: z.string(),
  channel: z.number(),
  node: z.string(),
  enabled: z.boolean(),
});

export const readTaskConfigZ = z
  .object({
    device: z.string(),
    sampleRate: z.number().min(0).max(1000),
    streamRate: z.number().min(0).max(200),
    channels: readTaskChannelConfigZ.array(),
  })
  .refine((c) => c.sampleRate >= c.streamRate, {
    path: ["streamRate"],
    message: "Stream rate must be lower than or equal to the sample rate",
  });

export type Device = device.Device<DeviceProperties>;
