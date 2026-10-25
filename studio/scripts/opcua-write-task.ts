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

const NODES = ["command_0", "command_1"];

/**
 * Docs `reference/hardware/opc-ua/write-task` (Create an OPC UA write task): create a
 * write task from a simulated OPC UA server, drag two writable variables from the
 * browser into the channels, and start the task.
 */
export default serverTaskScript(OPCUA, {
  item: "Create write task",
  channels: browseNodes(NODES),
});
