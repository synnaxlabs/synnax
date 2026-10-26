// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type Synnax } from "@synnaxlabs/client";

/** AUTOMATION names the Arc automation that runs the press sequence. */
export const AUTOMATION = "Press sequence";

/** createTankIndex creates `tank_time`, the index of the tank pressure. */
export const createTankIndex = async (client: Synnax): Promise<channel.Channel> =>
  await client.channels.create(
    { name: "tank_time", isIndex: true, dataType: "timestamp" },
    { retrieveIfNameExists: true },
  );

/**
 * channels returns the channels the press sequence commands and reads: the press and
 * vent valve commands, the start command, and `tank_pt` on `index`.
 */
export const channels = (index: channel.Key): channel.New[] => [
  { name: "press_vlv_cmd", dataType: "uint8", virtual: true },
  { name: "vent_vlv_cmd", dataType: "uint8", virtual: true },
  { name: "start_cmd", dataType: "uint8", virtual: true },
  { name: "tank_pt", dataType: "float64", index },
];
