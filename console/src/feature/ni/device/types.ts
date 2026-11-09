// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device, ni } from "@synnaxlabs/client";
import { z } from "zod";

import { Device } from "@/platform/device";

export const MAKE = "NI";

/** The name a rack lists in its integrations when its Driver runs this one. */
export const INTEGRATION = "ni";
export const makeZ = z.literal(MAKE);

export const propertiesZ = z.object({
  identifier: Device.identifierZ.catch(""),
  analogInput: z
    .object({
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default(() => ({})),
    })
    .default(() => ({ index: 0, channels: {} })),
  analogOutput: z
    .object({
      stateIndex: channel.keyZ.catch(0),
      channels: z.record(z.string(), Device.commandStatePairZ).default(() => ({})),
    })
    .default(() => ({ stateIndex: 0, channels: {} })),
  counterInput: z
    .object({
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default(() => ({})),
    })
    .default(() => ({ index: 0, channels: {} })),
  digitalInput: z
    .object({
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default(() => ({})),
    })
    .default(() => ({ index: 0, channels: {} })),
  digitalOutput: z
    .object({
      stateIndex: channel.keyZ.catch(0),
      channels: z.record(z.string(), Device.commandStatePairZ).default(() => ({})),
    })
    .default(() => ({ stateIndex: 0, channels: {} })),
  hardware: ni.hardwareZ.optional(),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  identifier: "",
  analogInput: { index: 0, channels: {} },
  analogOutput: { stateIndex: 0, channels: {} },
  counterInput: { index: 0, channels: {} },
  digitalInput: { index: 0, channels: {} },
  digitalOutput: { stateIndex: 0, channels: {} },
};

export interface Device extends device.Device<typeof propertiesZ, typeof makeZ> {}

export const SCHEMAS = {
  properties: propertiesZ,
  make: makeZ,
  model: z.string(),
} as const satisfies device.DeviceSchemas<typeof propertiesZ, typeof makeZ>;
