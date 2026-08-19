// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `console/users/register`: click the "+" action in the Users toolbar,
 * fill the form and pick a role, and submit; the new user appears under its
 * role in the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await capture.login(session, { username: "synnax", password: "seldon" }, project);

    await capture.openToolbar(session, "user");
    await capture.resizeToolbar(session, 400);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1000);

    await capture.clickToolbarCreate(session);
    const first = page.getByPlaceholder("Richard").first();
    await session.waitFor(first);
    await session.hold(600);

    await session.click(first);
    await session.type("Grace");
    await session.click(page.getByPlaceholder("Feynman").first());
    await session.type("Hopper");
    await session.click(page.getByPlaceholder("username").first());
    await session.type("grace");
    await session.click(page.getByPlaceholder("password").first());
    await session.type("flow-rate-9");
    await session.hold(400);

    await session.click(page.getByText("Select role", { exact: true }).first(), {
      text: true,
    });
    const role = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: "Operator" })
      .first();
    await session.waitFor(role);
    await session.click(role.getByText("Operator", { exact: true }).first(), {
      text: true,
    });
    await session.hold(600);

    await capture.clickButton(session, "Register");
    await session.waitForHidden(first);
    await session.hold(800);

    // The toolbar groups users by role and the groups start collapsed; expand
    // the Operator group to show the new user.
    const drawer = page.locator(".console-nav__drawer").first();
    await session.click(drawer.getByText("Operator", { exact: true }).first(), {
      text: true,
    });
    // Tree entries show the username, not the full name.
    const created = drawer.getByText("grace", { exact: true }).first();
    await session.waitFor(created);
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};
