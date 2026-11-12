// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { openProject } from "./common/desktop";

export const desktop = true;

/**
 * Docs `reference/client/connect` (Connect from Synnax Desktop): click the code button
 * in the top bar, copy the install command and the code of the Python tab, then open
 * the TypeScript tab, which shows the address and the API key, and copy its code.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openProject(session);
  await capture.clearPanel(session);
  await capture.hideBottomToolbar(session);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await session.click(page.locator("button:has(svg.lyra-icon--code)").first(), {
    zoom: false,
  });
  const modal = page.locator(capture.MODAL).first();
  await session.waitFor(modal.getByLabel("Python code", { exact: true }));
  await session.zoom(modal);
  await session.hold(1500);

  const copy = (label: string) =>
    modal.getByRole("button", { name: `Copy ${label}`, exact: true });
  await session.click(copy("Python install command"));
  await session.hold(1200);
  await session.click(copy("Python code"));
  await session.hold(1500);

  await session.click(modal.getByText("TypeScript", { exact: true }).first(), {
    text: true,
  });
  await session.waitFor(modal.getByLabel("TypeScript code", { exact: true }));
  await session.hold(1500);
  await session.click(copy("TypeScript code"));
  await session.hold(1500);
  session.endZoom();
  await session.moveTo({ x: 1100, y: 800 });
  await session.hold(2500);
};
