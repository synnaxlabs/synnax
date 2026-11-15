// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { openCreateKey } from "./common/credentials";

export const viewport = { width: 1280, height: 800 };

/**
 * Still for the API keys section of the v0.61 release notes: the credentials dialog of
 * a user with two API keys and the new key that the Core shows once.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const client = fixtures.connect();
  try {
    await client.auth.createKey({ name: "Sequence runner" });
    await client.auth.createKey({ name: "Nightly report" });
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await capture.openSettings(session, "Users");
    const item = page.getByRole("option", { name: "synnax", exact: true });
    await session.waitFor(item);
    await capture.contextMenu(session, item, "Manage sign-in");
    session.endZoom();
    const modal = page.locator(capture.MODAL).first();
    await session.waitFor(modal.getByText("Nightly report", { exact: true }));
    await openCreateKey(session, modal, "Test stand scripts");
    await session.moveTo({ x: 26, y: 600 });
    await session.settle(1200);
    await session.still("window");
  } finally {
    await client.close();
  }
};
