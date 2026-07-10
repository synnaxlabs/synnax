// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { createResource } from "@/platform/tree/testutil";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectTestProject,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

describe("schematic/search", () => {
  it("opens the schematic as a tab when the search result is selected", async () => {
    const key = uuid.create();
    const name = uniqueName("schematic");
    const resource = createResource(schematic.ontologyID(key), name);
    const SearchListItem = Schematic.SEARCH_LIST_ITEMS.schematic;
    if (SearchListItem == null)
      throw new Error("schematic SearchListItem is not defined");
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
    fireEvent.click(await screen.findByText(name), { detail: 0 });
    const tab = await resolveFocusedTab(store, client);
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    expect(tab.resource.key).toBe(key);
  });
});
