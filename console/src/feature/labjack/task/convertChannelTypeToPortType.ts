// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as Device from "@/feature/labjack/device/types";
import {
  type ReadChannelType,
  type WriteChannelType,
} from "@/feature/labjack/task/types";

const READ_CHANNEL_TO_PORT_MAP = {
  analog: Device.AI_PORT_TYPE,
  digital: Device.DI_PORT_TYPE,
  thermocouple: Device.AI_PORT_TYPE,
} as const satisfies Record<ReadChannelType, Device.PortType>;

export type ConvertReadChannelTypeToPortType = typeof READ_CHANNEL_TO_PORT_MAP;

export const convertReadChannelTypeToPortType = <T extends ReadChannelType>(
  type: T,
): ConvertReadChannelTypeToPortType[T] => READ_CHANNEL_TO_PORT_MAP[type];

// Thermocouple channels also live on AI ports, so the reverse mapping resolves an AI
// port to an analog channel.
const PORT_TO_READ_CHANNEL_MAP = {
  [Device.AI_PORT_TYPE]: "analog",
  [Device.DI_PORT_TYPE]: "digital",
} as const satisfies Record<Device.AIPortType | Device.DIPortType, ReadChannelType>;

export type ConvertPortTypeToReadChannelType = typeof PORT_TO_READ_CHANNEL_MAP;

export const convertPortTypeToReadChannelType = <
  T extends Device.AIPortType | Device.DIPortType,
>(
  type: T,
): ConvertPortTypeToReadChannelType[T] => PORT_TO_READ_CHANNEL_MAP[type];

const WRITE_CHANNEL_TO_PORT_MAP = {
  analog: Device.AO_PORT_TYPE,
  digital: Device.DO_PORT_TYPE,
} as const satisfies Record<WriteChannelType, Device.PortType>;

export type ConvertWriteChannelTypeToPortType = typeof WRITE_CHANNEL_TO_PORT_MAP;

export const convertWriteChannelTypeToPortType = <T extends WriteChannelType>(
  type: T,
): ConvertWriteChannelTypeToPortType[T] => WRITE_CHANNEL_TO_PORT_MAP[type];

const PORT_TO_WRITE_CHANNEL_MAP = {
  [Device.AO_PORT_TYPE]: "analog",
  [Device.DO_PORT_TYPE]: "digital",
} as const satisfies Record<Device.AOPortType | Device.DOPortType, WriteChannelType>;

export type ConvertPortTypeToWriteChannelType = typeof PORT_TO_WRITE_CHANNEL_MAP;

export const convertPortTypeToWriteChannelType = <
  T extends Device.AOPortType | Device.DOPortType,
>(
  type: T,
): ConvertPortTypeToWriteChannelType[T] => PORT_TO_WRITE_CHANNEL_MAP[type];
