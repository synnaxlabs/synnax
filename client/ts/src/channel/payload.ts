// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Density, Rate } from "@synnaxlabs/x";
import { z } from "zod";

export const channelPayloadSchema = z.object({
  key: z.string(),
  rate: z.number().transform((n) => new Rate(n)),
  dataType: z.string().transform((s) => new DataType(s)),
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

export const unkeyedChannelPayloadSchema = channelPayloadSchema.extend({
  key: z.string().optional().default(""),
});

export type UnkeyedChannelPayload = z.infer<typeof unkeyedChannelPayloadSchema>;
