// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { type CaptureSession } from "@/capture/rig";

/**
 * zoomCenter zooms `amount` deep on the center of a region too large for its fit to
 * zoom, and the camera pans with the cursor from there. Throws when the region is
 * hidden.
 */
export const zoomCenter = async (
  session: CaptureSession,
  region: Locator,
  amount: number,
): Promise<void> => {
  const box = await region.boundingBox();
  if (box == null) throw new Error("zoom region is hidden");
  await session.zoom({ x: box.x + box.width / 2, y: box.y + box.height / 2 }, amount);
};
