// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NI_DAQ, taskScript } from "./common/task";

/**
 * Docs `reference/hardware/tasks` (Ports): in an NI analog read task, select Ports above
 * the channel list, click a free port to add a channel on it, click a used port to
 * select its channel, go back to the list, and start the task.
 */
export default taskScript({
  device: NI_DAQ,
  item: "Create analog read task",
  channels: async (session, form) => {
    const view = (name: string) => form.getByText(name, { exact: true }).first();
    await session.click(view("Ports"), { text: true });
    const grid = form.getByRole("group", { name: `${NI_DAQ.name} ports` });
    await session.waitFor(grid);
    await session.hold(1500);
    const port = (index: number) => grid.locator("button").nth(index);
    await session.click(port(2));
    await session.hold(1000);
    await session.click(port(5));
    await session.hold(1000);
    await session.click(port(2));
    await session.hold(1500);
    await session.click(view("List"), { text: true });
    await session.hold(800);
  },
});
