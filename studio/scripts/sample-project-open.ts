// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { sampleLink, symbol } from "./common/sample";

export const desktop = true;

/**
 * Docs `get-started/sample-project/open`: on the empty projects screen of Synnax
 * Desktop, select Open sample project. The project opens on the Tank schematic, the
 * Tank pressure line plot, and the Press sequence.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  const link = sampleLink(page);
  await session.waitFor(link, 1800);
  await session.moveTo(capture.PARK);

  session.startRecording();
  await session.hold(1800);
  await session.zoom(page.locator(".console-project-splash__list").first());
  await session.hold(800);
  await session.click(link, { zoom: false });
  session.endZoom();
  await session.waitFor(symbol(page, "Start"), 1800);
  await session.moveTo(capture.PARK);
  await session.hold(3000);
};
