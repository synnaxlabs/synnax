// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addChannel, choose, NI_DAQ, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/ni/counter-read-task` (Create a counter read task): set the
 * rates, add a channel on the next counter, make it an edge count, and start the task.
 */
export default taskScript({
  device: NI_DAQ,
  item: "Create counter read task",
  rates: { "Sample rate": "100", "Stream rate": "25" },
  channels: async (session, form) => {
    await addChannel(session, form);
    await choose(session, form, "Channel type", "Edge count");
  },
});
