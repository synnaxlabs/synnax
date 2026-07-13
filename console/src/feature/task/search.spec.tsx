// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { type Tree } from "@/platform/tree";
import { createEntry } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createTask = async () => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await rack.createTask({
    name: uniqueName("task"),
    type: NI.Task.ANALOG_READ_TYPE,
    config: {},
  });
};

describe("task/search", () => {
  it("places the task's configuration layout when the search result is selected", async () => {
    const t = await createTask();
    const entry = createEntry(t.ontologyID, t.name);
    const SearchListItem = Task.SEARCH_LIST_ITEMS.task;
    if (SearchListItem == null) throw new Error("task SearchListItem is not defined");
    const Harness = (): ReactElement => {
      const staticProps = List.useStaticData<string, Tree.Entry>({
        data: [entry],
      });
      return (
        <Select.Frame<string, Tree.Entry>
          {...staticProps}
          value={undefined}
          onChange={() => {}}
        >
          <SearchListItem key={entry.key} itemKey={entry.key} index={0} />
        </Select.Frame>
      );
    };
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Harness />, { wrapper });
    fireEvent.click(await screen.findByText(t.name), { detail: 0 });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), t.key);
      expect(placed?.type).toBe(NI.Task.ANALOG_READ_TYPE);
    });
  });
});
