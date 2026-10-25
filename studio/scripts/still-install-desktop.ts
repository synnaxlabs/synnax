// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type capture } from "@/index";

import { sampleLink } from "./common/sample";

export const desktop = true;

export const viewport = { width: 1280, height: 800 };

/**
 * Still for docs page `reference/get-started/install-desktop`: the projects screen of
 * Synnax Desktop on first launch, with Open sample project and New project.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await session.waitFor(sampleLink(page), 1800);
  await session.waitFor(page.getByText("New project").first());
  // Clear of the link and the button, so neither shows its hover state.
  await session.moveTo({ x: 1240, y: 760 });
  await session.settle(1000);
  await session.still("projects");
};
