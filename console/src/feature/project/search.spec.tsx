// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ontology, project } from "@synnaxlabs/client";
import { List, Select } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

describe("project/search", () => {
  it("selects the project when the search result is selected", async () => {
    const p = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const resource = createResource(project.ontologyID(p.key), p.name);
    const SearchListItem = Project.SEARCH_LIST_ITEMS.project;
    if (SearchListItem == null)
      throw new Error("project SearchListItem is not defined");
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
    render(<Harness />, { wrapper });
    fireEvent.click(await screen.findByText(p.name), { detail: 0 });
    await waitFor(() =>
      expect(Session.Project.selectOptionalSelected(store.getState())).toBe(p.key),
    );
  });
});
