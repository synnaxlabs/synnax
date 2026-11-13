// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { createKey, openCredentials } from "./common/credentials";

/**
 * Docs `reference/enterprise/users` (Manage credentials): right-click a user on the
 * Users page of Settings and select Manage credentials, create an API key, then set a
 * new password.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  await fixtures.createUsers([
    { username: "grace", firstName: "Grace", lastName: "Hopper", role: "Viewer" },
  ]);
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);

  await capture.openSettings(session, "Users");
  const item = page.getByRole("option", { name: "grace", exact: true });
  await session.waitFor(item);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1800);

  const modal = await openCredentials(session, item);
  await createKey(session, modal, "Test stand scripts");

  await session.click(modal.getByPlaceholder("New password", { exact: true }));
  await session.type("correct-horse", 70);
  await session.hold(400);
  await session.click(modal.getByRole("button", { name: "Set password", exact: true }));
  await session.waitFor(modal.getByText("Successfully changed password"));
  await session.hold(2500);
  session.endZoom();
  await session.hold(1500);
};
