// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  AI_PORT_TYPE,
  type AIPortType,
  AO_PORT_TYPE,
  type AOPortType,
  DI_PORT_TYPE,
  type DIPortType,
  DO_PORT_TYPE,
  type DOPortType,
  type PortType,
} from "@/feature/labjack/device/types";
import {
  type ReadChannelType,
  type WriteChannelType,
} from "@/feature/labjack/task/types";

const READ_CHANNEL_TO_PORT_MAP = {
  analog: AI_PORT_TYPE,
  digital: DI_PORT_TYPE,
  thermocouple: AI_PORT_TYPE,
} as const satisfies Record<ReadChannelType, PortType>;

export type ConvertReadChannelTypeToPortType = typeof READ_CHANNEL_TO_PORT_MAP;

export const convertReadChannelTypeToPortType = <T extends ReadChannelType>(
  type: T,
): ConvertReadChannelTypeToPortType[T] => READ_CHANNEL_TO_PORT_MAP[type];

// Thermocouple channels also live on AI ports, so the reverse mapping resolves an AI
// port to an analog channel.
const PORT_TO_READ_CHANNEL_MAP = {
  [AI_PORT_TYPE]: "analog",
  [DI_PORT_TYPE]: "digital",
} as const satisfies Record<AIPortType | DIPortType, ReadChannelType>;

export type ConvertPortTypeToReadChannelType = typeof PORT_TO_READ_CHANNEL_MAP;

export const convertPortTypeToReadChannelType = <T extends AIPortType | DIPortType>(
  type: T,
): ConvertPortTypeToReadChannelType[T] => PORT_TO_READ_CHANNEL_MAP[type];

const WRITE_CHANNEL_TO_PORT_MAP = {
  analog: AO_PORT_TYPE,
  digital: DO_PORT_TYPE,
} as const satisfies Record<WriteChannelType, PortType>;

export type ConvertWriteChannelTypeToPortType = typeof WRITE_CHANNEL_TO_PORT_MAP;

export const convertWriteChannelTypeToPortType = <T extends WriteChannelType>(
  type: T,
): ConvertWriteChannelTypeToPortType[T] => WRITE_CHANNEL_TO_PORT_MAP[type];

const PORT_TO_WRITE_CHANNEL_MAP = {
  [AO_PORT_TYPE]: "analog",
  [DO_PORT_TYPE]: "digital",
} as const satisfies Record<AOPortType | DOPortType, WriteChannelType>;

export type ConvertPortTypeToWriteChannelType = typeof PORT_TO_WRITE_CHANNEL_MAP;

export const convertPortTypeToWriteChannelType = <T extends AOPortType | DOPortType>(
  type: T,
): ConvertPortTypeToWriteChannelType[T] => PORT_TO_WRITE_CHANNEL_MAP[type];
