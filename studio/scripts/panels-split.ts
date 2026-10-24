// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { createStand, splitRight } from "./common/panels";

/**
 * Docs `console/panels/split`: drag the table tab to the right third of the panel,
 * so the line plot and the table show side by side.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await splitRight(session, stand.table);
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};
