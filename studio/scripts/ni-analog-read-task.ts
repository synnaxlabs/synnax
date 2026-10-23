// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addChannel, choose, NI_DAQ, setNumber, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/ni/analog-read-task` (Create an analog read task): set the
 * rates, add a channel on the next port, make it a current channel, and start the task.
 */
export default taskScript({
  device: NI_DAQ,
  item: "Create analog read task",
  fill: async (session, form) => {
    await setNumber(session, form, "Sample rate", "100");
    await setNumber(session, form, "Stream rate", "25");
    await addChannel(session, form);
    await choose(session, form, "Channel type", "Current");
  },
});
