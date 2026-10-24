// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addChannel, LABJACK, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/labjack/write-task` (Create a write task): set the state
 * update rate, add two digital outputs, and start the task.
 */
export default taskScript({
  device: LABJACK,
  item: "Create write task",
  rates: { "State update rate": "20" },
  channels: async (session, form) => {
    await addChannel(session, form);
    await addChannel(session, form);
  },
});
