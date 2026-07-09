// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Task } from "@/feature/task";
import { createResource } from "@/platform/tree/testutil";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

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
  it("opens the task's configuration view when the search result is selected", async () => {
    const t = await createTask();
    const resource = createResource(t.ontologyID, t.name);
    const SearchListItem = Task.SEARCH_LIST_ITEMS.task;
    if (SearchListItem == null) throw new Error("task SearchListItem is not defined");
    const Harness = (): ReactElement => {
      const staticProps = List.useStaticData<string, ontology.Resource>({
        data: [resource],
      });
      return (
        <Select.Frame<string, ontology.Resource>
          {...staticProps}
          value={undefined}
          onChange={() => {}}
        >
          <SearchListItem key={resource.key} itemKey={resource.key} index={0} />
        </Select.Frame>
      );
    };
    const { wrapper, store } = await createConsoleWrapper({ client });
    await selectTestProject(store, client);
    render(<Harness />, { wrapper });
    fireEvent.click(await screen.findByText(t.name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe(NI.Task.ANALOG_READ_TYPE);
  });
});
