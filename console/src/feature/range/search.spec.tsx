// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { createTestRange } from "@/platform/range/testutil";
import { type Tree } from "@/platform/tree";
import { createEntry } from "@/platform/tree/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("range/search", () => {
  describe("SearchListItem", () => {
    it("renders the entry name and the time range retrieved from the cluster", async () => {
      const rng = await createTestRange(client);
      const entry = createEntry(ranger.ontologyID(rng.key), rng.name);
      const SearchListItem = Range.SEARCH_LIST_ITEMS.range;
      if (SearchListItem == null)
        throw new Error("range SearchListItem is not defined");
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
      const { wrapper } = await createConsoleWrapper({ client });
      const { container } = render(<Harness />, { wrapper });
      expect(await screen.findByText(rng.name)).toBeTruthy();
      // createTestRange builds a range starting now, which formats as "Today ...".
      await waitFor(() => expect(container.textContent).toContain("Today"));
    });
  });
});
