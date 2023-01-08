// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { DataType, Density, Rate } from "../telem";

export const channelPayloadSchema = z.object({
  rate: z.number().transform((n) => new Rate(n)),
  dataType: z.string().transform((s) => new DataType(s)),
  key: z.string().default("").optional(),
  name: z.string().default("").optional(),
  nodeId: z.number().default(0).optional(),
  density: z
    .number()
    .default(0)
    .transform((n) => new Density(n))
    .optional(),
  index: z.string().default("").optional(),
  isIndex: z.boolean().default(false).optional(),
});

export type ChannelPayload = z.infer<typeof channelPayloadSchema>;

export const keyedChannelPayloadSchema = channelPayloadSchema.extend({
  key: z.string(),
});

export type KeyedChannelPayload = z.infer<typeof keyedChannelPayloadSchema>;
