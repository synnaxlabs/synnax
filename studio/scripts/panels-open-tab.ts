// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { pickComponent } from "./common/create-component";
import { createStand } from "./common/panels";

/**
 * Docs `console/panels/open-tab`: click the add button at the right of the tab bar,
 * select Line plot in the Create component tab, and end on the new plot.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    await session.waitFor(capture.tab(session.page, "Create component"));
    await pickComponent(session, "Line plot", ".pluto-line-plot");
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};
