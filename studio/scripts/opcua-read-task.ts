// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { browseNodes, OPCUA, serverTaskScript } from "./common/server";

export const driver = true;

const NODES = ["my_float_0", "my_float_1"];

/**
 * Docs `reference/hardware/opc-ua/read-task` (Create an OPC UA read task): create a
 * read task from a simulated OPC UA server, set the rates, drag two variables from the
 * browser into the channels, and start the task.
 */
export default serverTaskScript(OPCUA, {
  item: "Create read task",
  rates: { "Sample rate": "100", "Stream rate": "20" },
  channels: browseNodes(NODES),
});
