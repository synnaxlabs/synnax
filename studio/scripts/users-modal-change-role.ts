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
 * Docs `console/users/modal-change-role`: right-click a user in the Users
 * toolbar, pick "Change role", assign a different role in the dialog, and show
 * the user move under it.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.seedUsers([
    { username: "grace", firstName: "Grace", lastName: "Hopper", role: "Viewer" },
  ]);
  const { page } = session;
  const project = `Docs Videos ${Date.now().toString(36)}`;
  await capture.login(session, { username: "synnax", password: "seldon" }, project);

  await capture.openToolbar(session, "user");
  await capture.resizeToolbar(session, 400);
  // The toolbar groups users by role and the groups start collapsed.
  const drawer = page.locator(".console-nav__drawer").first();
  await session.click(drawer.getByText("Viewer", { exact: true }).first(), {
    text: true,
  });
  const item = capture.treeItem(page, "user:", "grace");
  await session.waitFor(item);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1000);

  await capture.contextMenu(session, item, "Change role");
  const modal = page.locator(".console-modal").first();
  // The select opens on the current role rather than a placeholder.
  const role = modal.getByText("Viewer", { exact: true }).first();
  await session.waitFor(role);
  await session.hold(600);
  await session.zoom(modal);

  await session.click(role, { text: true });
  // The list order is not deterministic; filter through the auto-focused
  // search so the role is at the top before clicking it.
  await session.waitFor(page.getByPlaceholder("Search roles...").first());
  await session.hold(400);
  await session.type("Engineer");
  const option = page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
    .filter({ hasText: "Engineer" })
    .first();
  await session.waitFor(option);
  await session.hold(400);
  await session.click(option.getByText("Engineer", { exact: true }).first(), {
    text: true,
  });
  await session.hold(600);

  await capture.clickButton(session, "Assign");
  await session.waitForHidden(modal);
  session.endZoom();
  await session.hold(800);

  // No zoom: the whole tree stays in frame while the user moves groups.
  await session.click(drawer.getByText("Engineer", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitFor(capture.treeItem(page, "user:", "grace"));
  await session.hold(2200);
};
