// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fixtures } from "@/index";

import { CHANNEL, setAddress } from "./common/modbus";
import { addChannel, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/modbus/write-task` (Create a Modbus write task): right-click
 * a Modbus server in the devices toolbar and create a write task, add coils at
 * addresses 3 and 4, then start the task. The server is simulated.
 */
export default taskScript({
  device: fixtures.MODBUS_PLC,
  item: "Create write task",
  channels: async (session, form) => {
    await addChannel(session, form);
    await setAddress(session, form.locator(CHANNEL).first(), "3");
    // The new channel takes the next address.
    await addChannel(session, form);
  },
});
