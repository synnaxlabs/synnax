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
export const makeZ = z.literal(MAKE);
export type Make = z.infer<typeof makeZ>;

export const propertiesZ = z.object({
  identifier: Common.Device.identifierZ,
  analogInput: z.object({
    portCount: z.number(),
    index: channel.keyZ,
    channels: z.record(z.string(), channel.keyZ),
  }),
  analogOutput: z.object({
    portCount: z.number(),
    stateIndex: channel.keyZ,
    channels: z.record(z.string(), Common.Device.commandStatePairZ),
  }),
  counterInput: z.object({
    portCount: z.number(),
    index: channel.keyZ,
    channels: z.record(z.string(), channel.keyZ),
  }),
  digitalInputOutput: z.object({
    portCount: z.number(),
    lineCounts: z.array(z.number()),
  }),
  digitalInput: z.object({
    portCount: z.number(),
    lineCounts: z.array(z.number()),
    index: channel.keyZ,
    channels: z.record(z.string(), channel.keyZ),
  }),
  digitalOutput: z.object({
    portCount: z.number(),
    lineCounts: z.array(z.number()),
    stateIndex: channel.keyZ,
    channels: z.record(z.string(), Common.Device.commandStatePairZ),
  }),
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
