// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const scannedChannelZ = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.number(),
});

export const readChannelZ = z.object({
  key: z.string(),
  channel: z.number(),
  nodeId: z.number(),
});

export type ReadChannel = z.infer<typeof readChannelZ>;

export const readTaskConfigZ = z
  .object({
    device: z.string(),
    sampleRate: z.number().min(0).max(1000),
    streamRate: z.number().min(0).max(200),
    channels: readChannelZ.array(),
  })
  .refine((c) => c.sampleRate >= c.streamRate, {
    path: ["streamRate"],
    message: "Stream rate must be lower than or equal to the sample rate",
  });
