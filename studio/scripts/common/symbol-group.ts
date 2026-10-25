// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

/**
 * setup logs in, opens a schematic, and creates and opens a symbol group named "Test
 * stand", off camera. The create symbol action stays disabled while a built-in group is
 * selected, so a shot that creates a symbol starts here.
 */
export const setup = async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.selectToolbarTab(session, "Symbols");

  await session.click(capture.toolbarButton(page, "group", "add"), { zoom: false });
  const groupName = page.locator(capture.MODAL).getByPlaceholder("Name").first();
  await session.waitFor(groupName);
  await groupName.fill("Test stand");
  await session.settle(300);
  await capture.clickButton(session, "Save");
  await session.waitForHidden(groupName);
  const group = page
    .locator(".console-schematic__symbols-group-list [role='tab']")
    .filter({ hasText: "Test stand" })
    .first();
  await session.waitFor(group);
  // The strip clips tabs past its end under the group actions.
  await group.scrollIntoViewIfNeeded();
  await session.click(group, { text: true, zoom: false });
  await session.settle(600);
};
