// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { z } from "zod";

export const MODEL_KEYS = ["LJM_dtT4", "LJM_dtT7", "LJM_dtT8"] as const;
export const modelKeyZ = z.enum(MODEL_KEYS);
export type ModelKey = z.infer<typeof modelKeyZ>;

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
  AIN: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), z.number()),
  }),
  DIN: z.object({ portCount: z.number(), channels: z.record(z.string(), z.number()) }),
  AO: z.object({
    portCount: z.number(),
    channels: z.record(commandStatePairZ),
  }),
  DO: z.object({
    portCount: z.number(),
    channels: z.record(z.string(), commandStatePairZ),
  }),
});
export type Properties = z.infer<typeof propertiesZ>;

export const ZERO_PROPERTIES: Properties = {
  readIndex: 0,
  writeStateIndex: 0,
  identifier: "",
  AIN: {
    portCount: 0,
    channels: {},
  },
  AO: {
    portCount: 0,
    channels: {},
  },
  DIN: {
    portCount: 0,
    channels: {},
  },
  DO: {
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

export const ModelDirectory: Record<ModelKey, DeviceInfo> = {
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

export const getZeroProperties = (model: ModelKey): Properties => {
  const deviceInfo = ModelDirectory[model];
  return {
    readIndex: 0,
    writeStateIndex: 0,
    identifier: "",
    AIN: createProperty(deviceInfo.analogInput),
    AO: createProperty(deviceInfo.analogOutput),
    DIN: createProperty(deviceInfo.digitalInputOutput),
    DO: createProperty(deviceInfo.flexInputOutput),
  };
};

export const DI_CHANNEL_TYPE = "DI";
export const diChannelTypeZ = z.literal(DI_CHANNEL_TYPE);
export type DIChannelType = z.infer<typeof diChannelTypeZ>;
export const AO_CHANNEL_TYPE = "AO";
export const aoChannelTypeZ = z.literal(AO_CHANNEL_TYPE);
export type AOChannelType = z.infer<typeof aoChannelTypeZ>;
export const AI_CHANNEL_TYPE = "AI";
export const aiChannelTypeZ = z.literal(AI_CHANNEL_TYPE);
export type AIChannelType = z.infer<typeof aiChannelTypeZ>;
export const DO_CHANNEL_TYPE = "DO";
export const doChannelTypeZ = z.literal(DO_CHANNEL_TYPE);
export type DOChannelType = z.infer<typeof doChannelTypeZ>;

export const inputChannelTypeZ = z.union([diChannelTypeZ, aiChannelTypeZ]);
export type InputChannelType = z.infer<typeof inputChannelTypeZ>;
export const outputChannelTypeZ = z.union([aoChannelTypeZ, doChannelTypeZ]);
export type OutputChannelType = z.infer<typeof outputChannelTypeZ>;
export const channelTypeZ = z.union([inputChannelTypeZ, outputChannelTypeZ]);
export type ChannelType = z.infer<typeof channelTypeZ>;

export const aiPortZ = z.object({
  key: z.string(),
  type: aiChannelTypeZ,
  voltageRange: bounds.bounds,
});
export type AIPort = z.infer<typeof aiPortZ>;

export const diPortZ = z.object({
  key: z.string(),
  type: diChannelTypeZ,
});
export type DIPort = z.infer<typeof diPortZ>;

export const doPortZ = z.object({
  key: z.string(),
  type: doChannelTypeZ,
});
export type DOPort = z.infer<typeof doPortZ>;

export const aoPortZ = z.object({
  key: z.string(),
  type: aoChannelTypeZ,
});
export type AOPort = z.infer<typeof aoPortZ>;

export const portZ = z.union([aoPortZ, aiPortZ, doPortZ, diPortZ]);

export type Port = z.infer<typeof portZ>;

const aiFactory = (b: bounds.Bounds, voltageRange: bounds.Bounds): AIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => ({
    key: `AIN${i + b.lower}`,
    type: "AI",
    voltageRange,
  }));

const diFactory = (b: bounds.Bounds): DIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => ({
    key: `DIO${i + b.lower}`,
    type: "DI",
  }));

const doFactory = (b: bounds.Bounds): DOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => ({
    key: `DIO${i + b.lower}`,
    type: "DO",
  }));

const aoFactory = (b: bounds.Bounds): AOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => ({
    key: `DAC${i + b.lower}`,
    type: "AO",
  }));

const AIN_HIGH_VOLTAGE = bounds.construct(-10, 10);
const AIN_LOW_VOLTAGE = bounds.construct(0, 2.5);

export const portsZ = z.object({
  AO: aoPortZ.array(),
  DO: doPortZ.array(),
  AI: aiPortZ.array(),
  DI: diPortZ.array(),
});

export type Ports = z.infer<typeof portsZ>;

// T4

export const T4_AI_PORTS: AIPort[] = [
  ...aiFactory({ lower: 0, upper: 4 }, AIN_HIGH_VOLTAGE),
  ...aiFactory({ lower: 4, upper: 11 }, AIN_LOW_VOLTAGE),
];
export const T4_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
export const T4_DI_PORTS: DIPort[] = diFactory({ lower: 4, upper: 15 });
export const T4_DO_PORTS: DOPort[] = doFactory({ lower: 4, upper: 15 });
export const T4_PORTS: Ports = {
  AI: T4_AI_PORTS,
  AO: T4_AO_PORTS,
  DO: T4_DO_PORTS,
  DI: T4_DI_PORTS,
};

// T7

export const T7_AI_PORTS: AIPort[] = aiFactory(
  { lower: 0, upper: 13 },
  AIN_HIGH_VOLTAGE,
);
export const T7_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
export const T7_DI_PORTS: DIPort[] = diFactory({ lower: 0, upper: 22 });
export const T7_DO_PORTS: DOPort[] = doFactory({ lower: 0, upper: 22 });
export const T7_PORTS: Ports = {
  AI: T7_AI_PORTS,
  AO: T7_AO_PORTS,
  DI: T7_DI_PORTS,
  DO: T7_DO_PORTS,
};

// T8

export const T8_AI_PORTS: AIPort[] = aiFactory(
  { lower: 0, upper: 7 },
  AIN_HIGH_VOLTAGE,
);
export const T8_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
export const T8_DI_PORTS: DIPort[] = diFactory({ lower: 0, upper: 19 });
export const T8_DO_PORTS: DOPort[] = doFactory({ lower: 0, upper: 19 });
export const T8_PORTS: Ports = {
  AI: T8_AI_PORTS,
  AO: T8_AO_PORTS,
  DI: T8_DI_PORTS,
  DO: T8_DO_PORTS,
};

export const modelInfoZ = z.object({
  key: modelKeyZ,
  name: z.string(),
  ports: portsZ,
});

export interface ModelInfo extends z.infer<typeof modelInfoZ> {}

export const T4: ModelInfo = { key: "LJM_dtT4", name: "T4", ports: T4_PORTS };
export const T7: ModelInfo = { key: "LJM_dtT7", name: "T7", ports: T7_PORTS };
export const T8: ModelInfo = { key: "LJM_dtT8", name: "T8", ports: T8_PORTS };

export const devicesZ = z.object({
  LJM_dtT4: modelInfoZ,
  LJM_dtT7: modelInfoZ,
  LJM_dtT8: modelInfoZ,
});

export type Devices = z.output<typeof devicesZ>;

export const DEVICES: Devices = {
  LJM_dtT4: T4,
  LJM_dtT7: T7,
  LJM_dtT8: T8,
};
