// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addChannel, choose, taskScript, VALVE_OUTPUTS } from "./common/task";

/**
 * Docs `reference/hardware/ethercat/write-task` (Create a write task): set the state
 * update rate, add a channel, select the module and its output PDO, and start the task.
 */
export default taskScript({
  device: VALVE_OUTPUTS,
  item: "Create write task",
  rates: { "State update rate": "50" },
  channels: async (session, form) => {
    await addChannel(session, form);
    await choose(session, form, "Slave device", VALVE_OUTPUTS.name);
    await choose(session, form, "PDO", "Output");
  },
});
