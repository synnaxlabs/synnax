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
 * Docs `console/users/modal-change-role`: on the Users page of Settings, pick a
 * different role in a user's role select.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createUsers([
    { username: "grace", firstName: "Grace", lastName: "Hopper", role: "Viewer" },
  ]);
  const { page } = session;
  await capture.login(session);
  await capture.openSettings(session, "Users");
  const item = page.getByRole("option", { name: "grace", exact: true });
  await session.waitFor(item);
  await session.moveTo({ x: 756, y: 700 });

  session.startRecording();
  await session.hold(1800);

  await session.zoom(item);
  await session.click(item.getByText("Viewer", { exact: true }), { text: true });
  // The list order is not deterministic; filter through the auto-focused
  // search so the role is at the top before clicking it.
  await session.waitFor(page.getByPlaceholder("Search roles...").first());
  await session.hold(400);
  await session.type("Engineer");
  const option = page.getByRole("option", { name: /^Engineer/ }).first();
  await session.waitFor(option);
  await session.hold(400);
  await session.click(option, { text: true });
  await session.waitFor(item.getByText("Engineer", { exact: true }));
  session.endZoom();
  // Park clear of the component buttons, so none shows a hover state.
  await session.moveTo({ x: 1300, y: 780 });
  await session.hold(2800);
};
