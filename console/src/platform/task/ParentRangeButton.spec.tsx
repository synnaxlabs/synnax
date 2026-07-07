// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, panel, ranger, task } from "@synnaxlabs/client";
import { TimeRange } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  renderInTaskForm,
  renderInTaskFormWithClient,
  seedSelectedPanel,
} from "@/platform/task/testutil";
import { assertDefined, uniqueName } from "@/testutil";

describe("ParentRangeButton", () => {
  it("should render nothing when the task has no parent range", async () => {
    const { container } = await renderInTaskForm(<Task.ParentRangeButton />, {
      values: { key: undefined },
    });
    expect(container.textContent).toBe("");
  });

  describe("with a live client", () => {
    const client = createTestClient();

    const createTaskWithParentRange = async () => {
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "test_type",
        config: {},
      });
      const range = await client.ranges.create({
        name: uniqueName("rng"),
        timeRange: new TimeRange(1, 2),
      });
      await client.ontology.addChildren(
        ranger.ontologyID(range.key),
        task.ontologyID(tsk.key),
      );
      return { tsk, range };
    };

    it("should display the parent range the task is snapshotted to", async () => {
      const { tsk, range } = await createTaskWithParentRange();
      await renderInTaskFormWithClient(<Task.ParentRangeButton />, {
        client,
        values: { key: tsk.key },
      });
      await waitFor(() => expect(screen.getByText("Snapshotted to")).toBeTruthy());
      expect(screen.getByText(range.name)).toBeTruthy();
    });

    it("should open the parent range as a tab when clicked", async () => {
      const { tsk, range } = await createTaskWithParentRange();
      const { store, wrapper } = await renderInTaskFormWithClient(
        <Task.ParentRangeButton />,
        { client, values: { key: tsk.key } },
      );
      const seeded = await seedSelectedPanel(wrapper, store, client);
      fireEvent.click(await screen.findByText(range.name, {}));
      await waitFor(() => {
        const doc = seeded.fluxStore.panels.get(seeded.panelKey);
        assertDefined(doc, "panel doc missing from flux store");
        const tab = panel.findTabByResource(doc.root, ranger.ontologyID(range.key));
        assertDefined(tab, "range resource tab was not opened");
      });
    });
  });
});
