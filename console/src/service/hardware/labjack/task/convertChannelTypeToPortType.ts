// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/labjack/device";
import { type ChannelType } from "@/hardware/labjack/task/types";

const CHANNEL_TO_PORT_MAP = {
  AI: Device.AI_PORT_TYPE,
  AO: Device.AO_PORT_TYPE,
  DI: Device.DI_PORT_TYPE,
  DO: Device.DO_PORT_TYPE,
  TC: Device.AI_PORT_TYPE,
} as const satisfies Record<ChannelType, Device.PortType>;

export type ConvertChannelTypeToPortType = typeof CHANNEL_TO_PORT_MAP;

export const convertChannelTypeToPortType = <T extends ChannelType>(
  type: T,
): ConvertChannelTypeToPortType[T] => CHANNEL_TO_PORT_MAP[type];
