// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, schematic } from "@synnaxlabs/client";
import { List, Select } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { renderWithConsole, uniqueName } from "@/testutil";

describe("schematic/search", () => {
  it("places the schematic's layout when the search result is selected", async () => {
    const key = id.create();
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
    const { store } = await renderWithConsole(<Harness />);
    fireEvent.click(await screen.findByText(name), { detail: 0 });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), key);
      expect(placed?.type).toBe(Schematic.LAYOUT_TYPE);
      expect(placed?.name).toBe(name);
    });
  });
});
