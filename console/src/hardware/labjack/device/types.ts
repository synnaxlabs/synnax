// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const MODELS = ["LJM_dtT4", "LJM_dtT7", "LJM_dtT8"] as const;
export const modelZ = z.enum(MODELS);
export type Models = z.infer<typeof modelZ>;

const IDENTIFIER_MESSAGE = "Identifier must be between 2-12 characters";
const identifierZ = z.string().min(2, IDENTIFIER_MESSAGE).max(12, IDENTIFIER_MESSAGE);

export const configurablePropertiesZ = z.object({
  name: z.string().min(1, "Name must be at least 1 character long"),
  identifier: identifierZ,
});
export type ConfigurableProperties = z.infer<typeof configurablePropertiesZ>;

const commandStatePairZ = z.object({
  command: z.number(),
  state: z.number(),
});

export const propertiesZ = z.object({
  identifier: identifierZ,
  readIndex: z.number(),
  writeStateIndex: z.number(),
  analogInput: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), z.number()),
  }),
  analogOutput: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), commandStatePairZ),
  }),
  digitalInputOutput: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), z.number().or(commandStatePairZ)),
  }),
  flexInputOutput: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), z.number().or(commandStatePairZ)),
  }),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  readIndex: 0,
  writeStateIndex: 0,
  identifier: "",
  analogInput: {
    portCount: 0,
    channels: {},
  },
  analogOutput: {
    portCount: 0,
    channels: {},
  },
  digitalInputOutput: {
    portCount: 0,
    channels: {},
  },
  flexInputOutput: {
    portCount: 0,
    channels: {},
  },
};

type DeviceInfo = {
  analogInput: number;
  analogOutput: number;
  digitalInputOutput: number;
  flexInputOutput: number;
};

export const ModelDirectory: Record<Models, DeviceInfo> = {
  LJM_dtT4: {
    analogInput: 4,
    analogOutput: 2,
    digitalInputOutput: 8,
    flexInputOutput: 8,
  },
  LJM_dtT7: {
    analogInput: 14,
    analogOutput: 2,
    digitalInputOutput: 23,
    flexInputOutput: 0,
  },
  LJM_dtT8: {
    analogInput: 8,
    analogOutput: 2,
    digitalInputOutput: 20,
    flexInputOutput: 0,
  },
};

const createProperty = (portCount: number) => ({
  portCount,
  channels: {},
});

export const getZeroProperties = (model: Models): Properties => {
  const deviceInfo = ModelDirectory[model];
  return {
    readIndex: 0,
    writeStateIndex: 0,
    identifier: "",
    analogInput: createProperty(deviceInfo.analogInput),
    analogOutput: createProperty(deviceInfo.analogOutput),
    digitalInputOutput: createProperty(deviceInfo.digitalInputOutput),
    flexInputOutput: createProperty(deviceInfo.flexInputOutput),
  };
};
