// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AIChannel } from "@/feature/ni/task/types";

/** The key the built-in temperature sensor is stored under. It has no port. */
const BUILT_IN_TEMP_KEY = "boardTempSensor";

/**
 * Returns the key an analog input channel's Synnax channel is stored under in its
 * device's properties.
 */
export const getAIChannelDeviceKey = (channel: AIChannel): string =>
  channel.type === "ai_temp_builtin" ? BUILT_IN_TEMP_KEY : channel.port.toString();

/** Names an analog input channel that the user left unnamed. */
export const getAIChannelSuffix = (channel: AIChannel): string =>
  channel.type === "ai_temp_builtin" ? "temp" : channel.port.toString();
