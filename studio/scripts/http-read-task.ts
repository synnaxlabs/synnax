// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { HTTP, serverTaskScript } from "./common/server";

export const driver = true;

/**
 * Docs `reference/hardware/http/read-task` (Create an HTTP read task): create a read
 * task from a simulated HTTP server, set the rate, add a GET endpoint, add a field that
 * reads the temperature from its response, and start the task.
 */
export default serverTaskScript(HTTP, {
  item: "Create read task",
  rates: { Rate: "10" },
  channels: async (session, form) => {
    await session.click(form.getByText("Add endpoint", { exact: true }).first(), {
      text: true,
    });
    const path = form.getByPlaceholder("/api/data").first();
    await session.waitFor(path);
    await session.hold(500);
    await session.click(path);
    await session.type("/api/v1/data");
    await session.hold(600);
    await session.click(form.getByRole("button", { name: "Add field" }).first());
    const pointer = form.getByPlaceholder("/temperature").first();
    await session.waitFor(pointer);
    await session.hold(500);
    await session.click(pointer);
    await session.type("/temperature");
    await session.hold(300);
  },
});
