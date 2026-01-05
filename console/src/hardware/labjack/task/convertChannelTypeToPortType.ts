// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/labjack/device";
import {
  AI_CHANNEL_TYPE,
  AO_CHANNEL_TYPE,
  type ChannelType,
  DI_CHANNEL_TYPE,
  DO_CHANNEL_TYPE,
  TC_CHANNEL_TYPE,
} from "@/hardware/labjack/task/types";

const CHANNEL_TO_PORT_MAP = {
  [AI_CHANNEL_TYPE]: Device.AI_PORT_TYPE,
  [AO_CHANNEL_TYPE]: Device.AO_PORT_TYPE,
  [DI_CHANNEL_TYPE]: Device.DI_PORT_TYPE,
  [DO_CHANNEL_TYPE]: Device.DO_PORT_TYPE,
  [TC_CHANNEL_TYPE]: Device.AI_PORT_TYPE,
} as const;

export type ConvertChannelTypeToPortType = typeof CHANNEL_TO_PORT_MAP;

export const convertChannelTypeToPortType = <T extends ChannelType>(
  type: T,
): ConvertChannelTypeToPortType[T] => CHANNEL_TO_PORT_MAP[type];
