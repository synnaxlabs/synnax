// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type device } from "@synnaxlabs/client";
import { bounds, type record } from "@synnaxlabs/x";

import { type Common } from "@/hardware/common";

export const MAKE = "LabJack";
export type Make = typeof MAKE;

export const T4_MODEL = "LJM_dtT4";
export type T4Model = typeof T4_MODEL;

export const T7_MODEL = "LJM_dtT7";
export type T7Model = typeof T7_MODEL;

export const T8_MODEL = "LJM_dtT8";
export type T8Model = typeof T8_MODEL;

export type Model = T4Model | T7Model | T8Model;

export interface BasePort extends record.KeyedNamed<string> {
  alias?: string;
}

export const AI_PORT_TYPE = "AI";
export type AIPortType = typeof AI_PORT_TYPE;

export interface AIPort extends BasePort {
  type: AIPortType;
}

export const AIN_PORT_REGEX = /^AIN\d+$/;

export const AO_PORT_TYPE = "AO";
export type AOPortType = typeof AO_PORT_TYPE;

export interface AOPort extends BasePort {
  type: AOPortType;
}

export const DAC_PORT_REGEX = /^DAC\d+$/;

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

export const DIO_PORT_REGEX = /^DIO\d+$/;

export type Port = AIPort | AOPort | DIPort | DOPort;
export type PortType = Port["type"];

interface AltConfig {
  prefix: string;
  offset: number;
}

const convertAltConfigToAlias = ({ prefix, offset }: AltConfig, port: number): string =>
  `${prefix}${port - offset}`;

const aiFactory = (b: bounds.Bounds, altConfig?: AltConfig): AIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const key = `AIN${i + b.lower}`;
    return {
      key,
      name: key,
      type: AI_PORT_TYPE,
      alias: altConfig ? convertAltConfigToAlias(altConfig, i + b.lower) : undefined,
    };
  });

const aoFactory = (b: bounds.Bounds): AOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const key = `DAC${i + b.lower}`;
    return { key, name: key, type: AO_PORT_TYPE };
  });

const diFactory = (b: bounds.Bounds, altConfig: AltConfig): DIPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    const key = `DIO${port}`;
    return {
      key,
      name: key,
      type: DI_PORT_TYPE,
      alias: convertAltConfigToAlias(altConfig, port),
    };
  });

const doFactory = (b: bounds.Bounds, altConfig: AltConfig): DOPort[] =>
  Array.from({ length: bounds.span(b) + 1 }, (_, i) => {
    const port = i + b.lower;
    const key = `DIO${port}`;
    return {
      key,
      name: key,
      type: DO_PORT_TYPE,
      alias: convertAltConfigToAlias(altConfig, port),
    };
  });

interface PortsInfo {
  [AI_PORT_TYPE]: AIPort[];
  [AO_PORT_TYPE]: AOPort[];
  [DI_PORT_TYPE]: DIPort[];
  [DO_PORT_TYPE]: DOPort[];
}

const T4_AI_PORTS: AIPort[] = [
  ...aiFactory({ lower: 0, upper: 3 }),
  ...aiFactory({ lower: 4, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...aiFactory({ lower: 8, upper: 11 }, { prefix: "EIO", offset: 8 }),
];
const T4_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T4_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 4, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...diFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...diFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
  ...diFactory({ lower: 20, upper: 22 }, { prefix: "MIO", offset: 20 }),
];
const T4_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 4, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...doFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...doFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
  ...doFactory({ lower: 20, upper: 22 }, { prefix: "MIO", offset: 20 }),
];
const T4_PORTS: PortsInfo = {
  [AI_PORT_TYPE]: T4_AI_PORTS,
  [AO_PORT_TYPE]: T4_AO_PORTS,
  [DI_PORT_TYPE]: T4_DI_PORTS,
  [DO_PORT_TYPE]: T4_DO_PORTS,
};

const T7_AI_PORTS: AIPort[] = [
  ...aiFactory({ lower: 0, upper: 13 }),
  ...aiFactory({ lower: 48, upper: 127 }), //Channels for use with LabJack MUX80
];
const T7_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T7_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 0, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...diFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...diFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
  ...diFactory({ lower: 20, upper: 22 }, { prefix: "MIO", offset: 20 }),
];
const T7_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 0, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...doFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...doFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
  ...doFactory({ lower: 20, upper: 22 }, { prefix: "MIO", offset: 20 }),
];
const T7_PORTS: PortsInfo = {
  [AI_PORT_TYPE]: T7_AI_PORTS,
  [AO_PORT_TYPE]: T7_AO_PORTS,
  [DI_PORT_TYPE]: T7_DI_PORTS,
  [DO_PORT_TYPE]: T7_DO_PORTS,
};

const T8_AI_PORTS: AIPort[] = aiFactory({ lower: 0, upper: 7 });
const T8_AO_PORTS: AOPort[] = aoFactory({ lower: 0, upper: 1 });
const T8_DI_PORTS: DIPort[] = [
  ...diFactory({ lower: 0, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...diFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...diFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
];
const T8_DO_PORTS: DOPort[] = [
  ...doFactory({ lower: 0, upper: 7 }, { prefix: "FIO", offset: 0 }),
  ...doFactory({ lower: 8, upper: 15 }, { prefix: "EIO", offset: 8 }),
  ...doFactory({ lower: 16, upper: 19 }, { prefix: "CIO", offset: 16 }),
];
const T8_PORTS: PortsInfo = {
  [AI_PORT_TYPE]: T8_AI_PORTS,
  [AO_PORT_TYPE]: T8_AO_PORTS,
  [DI_PORT_TYPE]: T8_DI_PORTS,
  [DO_PORT_TYPE]: T8_DO_PORTS,
};

export interface Ports extends Record<Model, PortsInfo> {}
export const PORTS: Ports = {
  [T4_MODEL]: T4_PORTS,
  [T7_MODEL]: T7_PORTS,
  [T8_MODEL]: T8_PORTS,
};

export type Properties = {
  identifier: Common.Device.Identifier;
  readIndex: channel.Key;
  thermocoupleIndex: channel.Key;
  writeStateIndex: channel.Key;
  [AI_PORT_TYPE]: { channels: Record<string, channel.Key> };
  [AO_PORT_TYPE]: { channels: Record<string, Common.Device.CommandStatePair> };
  [DI_PORT_TYPE]: { channels: Record<string, channel.Key> };
  [DO_PORT_TYPE]: { channels: Record<string, Common.Device.CommandStatePair> };
};

export const ZERO_PROPERTIES: Properties = {
  identifier: "",
  readIndex: 0,
  thermocoupleIndex: 0,
  writeStateIndex: 0,
  [AI_PORT_TYPE]: { channels: {} },
  [AO_PORT_TYPE]: { channels: {} },
  [DI_PORT_TYPE]: { channels: {} },
  [DO_PORT_TYPE]: { channels: {} },
};

export interface Device extends device.Device<Properties, Make, Model> {}
export interface New extends device.New<Properties, Make, Model> {}
