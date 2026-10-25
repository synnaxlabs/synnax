// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fixtures } from "@/index";

import { CHANNEL, choose, setAddress } from "./common/modbus";
import { addChannel, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/modbus/read-task` (Create a Modbus read task): right-click
 * a Modbus server in the devices toolbar and create a read task, set the rates to 10 Hz
 * and 5 Hz, add a float32 holding register at address 0 and a coil at address 3, then
 * start the task. The server is simulated, so the task reads no values.
 */
export default taskScript({
  device: fixtures.MODBUS_PLC,
  item: "Create read task",
  rates: { "Sample rate": "10", "Stream rate": "5" },
  channels: async (session, form) => {
    const channels = form.locator(CHANNEL);
    await addChannel(session, form);
    await choose(session, channels.nth(0), "Coil", "Holding register");
    await choose(session, channels.nth(0), "uint8", "float32");
    await addChannel(session, form);
    await choose(session, channels.nth(1), "Holding register", "Coil");
    await setAddress(session, channels.nth(1), "3");
  },
});
