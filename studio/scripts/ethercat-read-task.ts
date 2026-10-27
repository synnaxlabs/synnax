// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addChannel, choose, TANK_PRESSURES, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/read-task` (Create a read task): set the rates, add
 * a channel, select the module and its first Value entry, and start the task.
 */
export default taskScript({
  device: TANK_PRESSURES,
  item: "Create read task",
  rates: { "Sample rate": "1000", "Stream rate": "25" },
  channels: async (session, form) => {
    await addChannel(session, form);
    await choose(session, form, "Slave device", TANK_PRESSURES.name);
    await choose(session, form, "Entry", "Value");
  },
});
