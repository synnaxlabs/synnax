// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";
import { bounds } from "@synnaxlabs/x";

import { type Common } from "@/hardware/common";

// Makes

export const MAKE = "LabJack";
export type Make = typeof MAKE;

// Model Keys

const T4_MODEL_KEY = "LJM_dtT4";
type T4ModelKey = typeof T4_MODEL_KEY;

const T7_MODEL_KEY = "LJM_dtT7";
type T7ModelKey = typeof T7_MODEL_KEY;

const T8_MODEL_KEY = "LJM_dtT8";
type T8ModelKey = typeof T8_MODEL_KEY;

export type ModelKey = T4ModelKey | T7ModelKey | T8ModelKey;

interface BasePort {
  key: string;
  aliases: string[];
}

export const AI_PORT_TYPE = "AI";
export type AIPortType = typeof AI_PORT_TYPE;

export interface AIPort extends BasePort {
  type: AIPortType;
  voltageRange: bounds.Bounds;
}

export const AO_PORT_TYPE = "AO";
export type AOPortType = typeof AO_PORT_TYPE;

export interface AOPort extends BasePort {
  type: AOPortType;
}

export const DI_PORT_TYPE = "DI";
export type DIPortType = typeof DI_PORT_TYPE;

export interface DIPort extends BasePort {
  type: DIPortType;
}

export const DO_PORT_TYPE = "DO";
export type DOPortType = typeof DO_PORT_TYPE;

export interface DOPort extends BasePort {
  type: DOPortType;
}

export type Port = AOPort | AIPort | DOPort | DIPort;
export type PortType = Port["type"];

interface AltConfig {
  prefix: string;
  offset: number;
}

const mapAltConfigsToAliases = (altConfigs: AltConfig[], port: number): string[] =>
  altConfigs.map((config) => `${config.prefix}${port - config.offset}`);

const aiFactory = (
  b: bounds.Bounds,
  voltageRange: bounds.Bounds,
  altConfigs: AltConfig[] = [],
): AIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    return {
      key: `AIN${port}`,
      type: AI_PORT_TYPE,
      voltageRange,
      aliases: mapAltConfigsToAliases(altConfigs, port),
    };
  });

const diFactory = (b: bounds.Bounds, altConfigs: AltConfig[] = []): DIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    return {
      key: `DIO${port}`,
      type: DI_PORT_TYPE,
      aliases: mapAltConfigsToAliases(altConfigs, port),
    };
  });

const doFactory = (b: bounds.Bounds, altConfigs: AltConfig[] = []): DOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    return {
      key: `DIO${port}`,
      type: DO_PORT_TYPE,
      aliases: mapAltConfigsToAliases(altConfigs, port),
    };
  });

const aoFactory = (b: bounds.Bounds, altConfigs: AltConfig[] = []): AOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    return {
      key: `DAC${port}`,
      type: AO_PORT_TYPE,
      aliases: mapAltConfigsToAliases(altConfigs, port),
    };
  });

const AIN_HIGH_VOLTAGE = bounds.construct(-10, 10);
const AIN_LOW_VOLTAGE = bounds.construct(0, 2.5);

interface Ports {
  AO: AOPort[];
  DO: DOPort[];
  AI: AIPort[];
  DI: DIPort[];
}

// T4

const T4_AI_PORTS: AIPort[] = [
  ...aiFactory({ lower: 0, upper: 4 }, AIN_HIGH_VOLTAGE),
  ...aiFactory({ lower: 5, upper: 11 }, AIN_LOW_VOLTAGE),
];
const T4_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T4_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 4, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...diFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
];
const T4_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 4, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...doFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
];

const T4_PORTS: Ports = {
  AI: T4_AI_PORTS,
  AO: T4_AO_PORTS,
  DO: T4_DO_PORTS,
  DI: T4_DI_PORTS,
};

// T7

const T7_AI_PORTS: AIPort[] = aiFactory({ lower: 0, upper: 13 }, AIN_HIGH_VOLTAGE);
const T7_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T7_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 0, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...diFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
  ...diFactory({ lower: 16, upper: 19 }, [{ prefix: "CIO", offset: 16 }]),
  ...diFactory({ lower: 20, upper: 22 }, [{ prefix: "MIO", offset: 20 }]),
];
const T7_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 0, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...doFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
  ...doFactory({ lower: 16, upper: 19 }, [{ prefix: "CIO", offset: 16 }]),
  ...doFactory({ lower: 20, upper: 22 }, [{ prefix: "MIO", offset: 20 }]),
];
const T7_PORTS: Ports = {
  AI: T7_AI_PORTS,
  AO: T7_AO_PORTS,
  DI: T7_DI_PORTS,
  DO: T7_DO_PORTS,
};

// T8

const T8_AI_PORTS: AIPort[] = aiFactory({ lower: 0, upper: 7 }, AIN_HIGH_VOLTAGE);
const T8_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T8_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 0, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...diFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
  ...diFactory({ lower: 16, upper: 19 }, [{ prefix: "CIO", offset: 16 }]),
];
const T8_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 0, upper: 7 }, [{ prefix: "FIO", offset: 0 }]),
  ...doFactory({ lower: 8, upper: 15 }, [{ prefix: "EIO", offset: 8 }]),
  ...doFactory({ lower: 16, upper: 19 }, [{ prefix: "CIO", offset: 16 }]),
];
const T8_PORTS: Ports = {
  AI: T8_AI_PORTS,
  AO: T8_AO_PORTS,
  DI: T8_DI_PORTS,
  DO: T8_DO_PORTS,
};

interface ModelInfo {
  key: ModelKey;
  name: string;
  ports: Ports;
}

const T4: ModelInfo = { key: T4_MODEL_KEY, name: "T4", ports: T4_PORTS };
const T7: ModelInfo = { key: T7_MODEL_KEY, name: "T7", ports: T7_PORTS };
const T8: ModelInfo = { key: T8_MODEL_KEY, name: "T8", ports: T8_PORTS };

interface Devices extends Record<ModelKey, ModelInfo> {}

export const DEVICES: Devices = {
  [T4_MODEL_KEY]: T4,
  [T7_MODEL_KEY]: T7,
  [T8_MODEL_KEY]: T8,
};

export type Properties = {
  identifier: Common.Device.Identifier;
  readIndex: channel.Key;
  thermocoupleIndex: channel.Key;
  writeStateIndex: channel.Key;
  [AI_PORT_TYPE]: { channels: Record<string, channel.Key> };
  [DI_PORT_TYPE]: { channels: Record<string, channel.Key> };
  [AO_PORT_TYPE]: { channels: Record<string, Common.Device.CommandStatePair> };
  [DO_PORT_TYPE]: { channels: Record<string, Common.Device.CommandStatePair> };
};

export const ZERO_PROPERTIES: Properties = {
  readIndex: 0,
  thermocoupleIndex: 0,
  writeStateIndex: 0,
  identifier: "",
  [AI_PORT_TYPE]: { channels: {} },
  [AO_PORT_TYPE]: { channels: {} },
  [DI_PORT_TYPE]: { channels: {} },
  [DO_PORT_TYPE]: { channels: {} },
};

export type Device = device.Device<Properties, Make, ModelKey>;
