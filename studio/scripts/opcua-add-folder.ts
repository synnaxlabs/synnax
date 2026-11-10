// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { OPCUA, serverTaskScript } from "./common/server";

export const driver = true;
export const integrations = ["opc"];

/**
 * Docs `reference/hardware/opc-ua/read-task` (Add a folder): drag a folder of the
 * simulated server into Channels and confirm the count of variables.
 */
export default serverTaskScript(OPCUA, {
  item: "Create read task",
  // The folder holds arrays, which a scalar task reports as bad values.
  unstarted: true,
  channels: async (session, form) => {
    const { page } = session;
    const folder = form
      .locator(".console-opc-browser")
      .first()
      .getByRole("treeitem")
      .filter({ hasText: /^MyObject$/ });
    await session.waitFor(folder);
    await session.hold(1000);
    const list = form.locator(".console-channel-list").first();
    await session.drag(folder, list);
    const modal = page.locator(capture.MODAL).first();
    const add = modal.getByRole("button", { name: "Add", exact: true });
    await session.waitFor(add, 1200);
    await session.zoom(modal);
    await session.hold(1800);
    await session.click(add);
    session.endZoom();
    await session.waitForHidden(modal);
    await capture.zoomCenter(
      session,
      form.locator(".console-task-channel-form-container"),
      1.35,
    );
    await session.waitFor(list.getByRole("option").first());
    await session.hold(2000);
  },
});
