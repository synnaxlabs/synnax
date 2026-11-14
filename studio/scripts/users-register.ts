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
 * Docs `console/users/register`: click "Create user" on the Users page of Settings,
 * fill the form and pick a role, and submit; the new user appears in the table.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.openSettings(session, "Users");
    await session.moveTo({ x: 756, y: 700 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickButton(session, "Create user");
    const name = page.getByPlaceholder("Richard Feynman").first();
    await session.waitFor(name);
    await session.hold(600);
    await session.zoom(page.locator(capture.MODAL).first());

    await session.click(name);
    await session.type("Grace Hopper");
    await session.click(page.getByPlaceholder("username").first());
    await session.type("grace");
    await session.click(page.getByPlaceholder("password").first());
    await session.type("flow-rate-9");
    await session.hold(400);

    await session.click(page.getByText("Select role", { exact: true }).first(), {
      text: true,
    });
    // The list order is not deterministic; filter through the auto-focused
    // search so the role is at the top before clicking it.
    await session.waitFor(page.getByPlaceholder("Search roles...").first());
    await session.hold(400);
    await session.type("Operator");
    const role = page.getByRole("option", { name: /^Operator/ }).first();
    await session.waitFor(role);
    await session.hold(400);
    await session.click(role, { text: true });
    await session.hold(600);

    await capture.clickButton(session, "Register");
    await session.waitForHidden(name);
    session.endZoom();
    await session.hold(800);

    await session.waitFor(
      page.getByRole("option", { name: "Grace Hopper", exact: true }),
    );
    // Park clear of the component buttons, so none shows a hover state.
    await session.moveTo({ x: 1300, y: 780 });
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};
