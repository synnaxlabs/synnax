// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fixtures } from "@/index";

import { CORE_RACK } from "./common/server";
import { ASSET, CONNECTION, siftScript } from "./common/sift";
import { choose, recordTask } from "./common/task";

export const desktop = true;

const CHANNELS = ["ox_pt_1", "ox_pt_2", "ox_tc_1"];
const LABEL = "Coldflow";

/**
 * Docs `reference/ranges/sift` (Stream live data): create a stream task from the Sift
 * connection in the devices toolbar, pick the asset, three live channels, and the label
 * that makes a range a Sift run, and start the task.
 */
export default siftScript(async (session) => {
  await fixtures.resetLabels([{ name: LABEL, color: "#3B7DDB" }]);
  const { page } = session;
  await recordTask(session, {
    device: CONNECTION,
    rack: CORE_RACK,
    item: "Create stream task",
    properties: async (session, form) => {
      await choose(session, form, "Asset", ASSET);
    },
    channels: async (session, form) => {
      const select = form.getByText("Select channels to send").first();
      await session.click(select, { text: true });
      await session.hold(500);
      await session.type("ox_");
      for (const name of CHANNELS) {
        const option = page.getByRole("option", { name, exact: true }).first();
        await session.waitFor(option);
        await session.click(option, { text: true });
        await session.hold(250);
      }
      await page.keyboard.press("Escape");
      await session.hold(500);
      await choose(session, form, "Labels", LABEL);
      await page.keyboard.press("Escape");
    },
  });
});
