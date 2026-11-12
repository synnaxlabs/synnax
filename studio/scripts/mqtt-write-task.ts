// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { openDiscover } from "./common/mqtt";
import { MQTT, serverTaskScript } from "./common/server";
import { choose, setText } from "./common/task";

export const driver = true;
export const integrations = ["modbus"];

/**
 * Docs `reference/hardware/mqtt/write-task` (Create an MQTT write task): open Add
 * targets, add a command topic by name, because a command topic publishes nothing until
 * a command arrives, set its pointer and JSON type, add a static field, and start the
 * task.
 */
export default serverTaskScript(MQTT, {
  item: "Create write task",
  channels: async (session, form) => {
    const dialog = await openDiscover(session, form, "Add targets");
    await session.waitFor(dialog.getByRole("treeitem").first());
    await session.hold(1500);
    await session.click(
      dialog.getByRole("button", { name: "Add a topic by name", exact: true }),
      { text: true },
    );
    session.endZoom();
    await session.waitForHidden(dialog);
    await capture.zoomCenter(
      session,
      form.locator(".console-task-channel-form-container"),
      1.35,
    );
    await session.hold(800);

    await setText(session, form, "Topic", "plant/valve/set");
    await setText(session, form, "Pointer", "/position");
    await choose(session, form, "JSON type", "Number");
    await session.click(
      form.getByRole("button", { name: "Static", exact: true }).first(),
    );
    await session.hold(600);
    const pointer = form.getByPlaceholder("/field", { exact: true }).first();
    await session.waitFor(pointer);
    await pointer.scrollIntoViewIfNeeded();
    await session.settle(300);
    await session.click(pointer);
    await session.type("/source");
    await session.hold(300);
    await session.click(form.getByPlaceholder("value", { exact: true }).first());
    await session.type("synnax");
    await session.hold(600);
  },
});
