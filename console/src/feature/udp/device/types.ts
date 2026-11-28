// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, udp } from "@synnaxlabs/client";
import { z } from "zod";

export const MAKE = "udp";

/** The name a rack lists in its integrations when its Driver runs this one. */
export const INTEGRATION_KEY = "udp";
const makeZ = z.literal(MAKE);
const modelZ = z.literal(MAKE);

export const ZERO_PROPERTIES: udp.Properties = udp.propertiesZ.parse({});

export interface Device extends device.Device<
  typeof udp.propertiesZ,
  typeof makeZ,
  typeof modelZ
> {}

export const SCHEMAS = {
  properties: udp.propertiesZ,
  make: makeZ,
  model: modelZ,
} as const satisfies device.DeviceSchemas<
  typeof udp.propertiesZ,
  typeof makeZ,
  typeof modelZ
>;
