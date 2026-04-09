// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type device } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const MAKE = "NI";
export type Make = typeof MAKE;
export const makeZ = z.literal(MAKE);

const ZERO_AI = { portCount: 0, index: 0, channels: {} };
const ZERO_AO = { portCount: 0, stateIndex: 0, channels: {} };
const ZERO_CI = { portCount: 0, index: 0, channels: {} };
const ZERO_DIO = { portCount: 0, lineCounts: [] as number[] };
const ZERO_DI = { portCount: 0, lineCounts: [] as number[], index: 0, channels: {} };
const ZERO_DO = {
  portCount: 0,
  lineCounts: [] as number[],
  stateIndex: 0,
  channels: {},
};

export const propertiesZ = z.object({
  identifier: Common.Device.identifierZ.catch(""),
  analogInput: z
    .object({
      portCount: z.number().default(0),
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default({}),
    })
    .default(ZERO_AI),
  analogOutput: z
    .object({
      portCount: z.number().default(0),
      stateIndex: channel.keyZ.catch(0),
      channels: z.record(z.string(), Common.Device.commandStatePairZ).default({}),
    })
    .default(ZERO_AO),
  counterInput: z
    .object({
      portCount: z.number().default(0),
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default({}),
    })
    .default(ZERO_CI),
  digitalInputOutput: z
    .object({
      portCount: z.number().default(0),
      lineCounts: z.array(z.number()).default([]),
    })
    .default(ZERO_DIO),
  digitalInput: z
    .object({
      portCount: z.number().default(0),
      lineCounts: z.array(z.number()).default([]),
      index: channel.keyZ.catch(0),
      channels: z.record(z.string(), channel.keyZ).default({}),
    })
    .default(ZERO_DI),
  digitalOutput: z
    .object({
      portCount: z.number().default(0),
      lineCounts: z.array(z.number()).default([]),
      stateIndex: channel.keyZ.catch(0),
      channels: z.record(z.string(), Common.Device.commandStatePairZ).default({}),
    })
    .default(ZERO_DO),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  identifier: "",
  analogInput: { portCount: 0, index: 0, channels: {} },
  analogOutput: { portCount: 0, stateIndex: 0, channels: {} },
  counterInput: { portCount: 0, index: 0, channels: {} },
  digitalInputOutput: { portCount: 0, lineCounts: [] },
  digitalInput: { portCount: 0, lineCounts: [], index: 0, channels: {} },
  digitalOutput: { portCount: 0, lineCounts: [], stateIndex: 0, channels: {} },
};

export interface Device extends device.Device<typeof propertiesZ, typeof makeZ> {}
export interface New extends device.New<typeof propertiesZ, typeof makeZ> {}

export const SCHEMAS = {
  properties: propertiesZ,
  make: makeZ,
  model: z.string(),
} as const satisfies device.DeviceSchemas<typeof propertiesZ, typeof makeZ>;
