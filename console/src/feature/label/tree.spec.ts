// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Label } from "@/feature/label";
import { Range } from "@/feature/range";
import { Status } from "@/feature/status";
import { createTestRange } from "@/platform/range/testutil";
import { Tree } from "@/platform/tree";
import {
  createHostItems,
  expandTreeRow,
  findTreeRow,
  HOST_ID,
  renderLiveTree,
} from "@/platform/tree/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("label/tree", () => {
  describe("children", () => {
    it("should list the ranges and statuses that hold the label", async () => {
      const lbl = await client.labels.create({
        name: uniqueName("label"),
        color: color.construct("#FF0000"),
      });
      const unlabeled = await createTestRange(client);
      const rng = await client.ranges.create({
        name: uniqueName("range"),
        timeRange: unlabeled.timeRange,
        labels: [lbl.key],
      });
      const st = await client.statuses.set({
        name: uniqueName("status"),
        variant: "info",
        message: "labeled",
        labels: [lbl.key],
      });
      await renderLiveTree({
        client,
        root: HOST_ID,
        items: {
          ...Label.TREE_ITEMS,
          ...Range.TREE_ITEMS,
          ...Status.TREE_ITEMS,
          ...createHostItems((c) => [
            Tree.createSource(c.labels, { keys: [lbl.key] }, Tree.toResource("label")),
          ]),
        },
      });
      await findTreeRow(lbl.name);
      expandTreeRow(lbl.name);
      expect(await findTreeRow(rng.name)).toBeTruthy();
      expect(await findTreeRow(st.name)).toBeTruthy();
      expect(screen.queryByText(unlabeled.name)).toBeNull();
    });
  });
});
