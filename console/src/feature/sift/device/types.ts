// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, sift } from "@synnaxlabs/client";
import { z } from "zod";

export const MAKE = "sift";
export const INTEGRATION = "sift";
const makeZ = z.literal(MAKE);
const modelZ = z.literal("sift");

const propertiesZ = sift.propertiesZ.extend({
  uri: z.string().min(1, "URI is required"),
  apiKey: z.uuid({ error: "API key is required" }),
});

export interface Properties extends z.infer<typeof propertiesZ> {}

export const ZERO_PROPERTIES: Properties = { uri: "", apiKey: "" };

export interface Device extends device.Device<
  typeof propertiesZ,
  typeof makeZ,
  typeof modelZ
> {}

export const SCHEMAS = {
  properties: propertiesZ,
  make: makeZ,
  model: modelZ,
} as const satisfies device.DeviceSchemas<
  typeof propertiesZ,
  typeof makeZ,
  typeof modelZ
>;
