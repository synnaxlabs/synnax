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

import * as v0 from "@/hardware/opc/device/types/v0";

const newPropertiesZ = v0.propertiesZ.omit({ read: true }).extend({
  read: z.object({
    indexes: z.number().array(),
    channels: z.record(z.string(), z.number()),
  }),
  version: z.literal("1.0.0"),
});

export const propertiesZ = newPropertiesZ.passthrough().or(
  v0.propertiesZ.passthrough().transform((v) => ({
    ...v,
    read: { indexes: [v.read.index], channels: v.read.channels },
    version: "1.0.0" as const,
  })),
);

export type Properties = z.infer<typeof propertiesZ>;
export type Device = device.Device<Properties>;
