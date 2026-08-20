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
 * Docs `console/users/drag-change-role`: drag a user node in the Users toolbar
 * onto a different role node and drop it; the user is reassigned to that role.
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
  const user = capture.treeItem(page, "user:", "grace");
  await session.waitFor(user);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1200);

  // No zoom: the source row and the target role must both stay in frame.
  await session.drag(user, capture.treeItem(page, "role:", "Engineer"), {
    zoom: false,
  });
  await session.hold(1400);

  await session.click(drawer.getByText("Engineer", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitFor(capture.treeItem(page, "user:", "grace"));
  await session.hold(2400);
};
