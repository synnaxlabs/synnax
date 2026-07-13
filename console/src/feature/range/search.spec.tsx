// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List, Select } from "@synnaxlabs/pluto";
import { screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import { renderWithConsole } from "@/testutil";

const client = createTestClient();

describe("range/search", () => {
  describe("SearchListItem", () => {
    it("renders the resource name and time range", async () => {
      const rng = await createTestRange(client);
      const resource = createResource(ranger.ontologyID(rng.key), rng.name, {
        timeRange: rng.timeRange.numeric,
      });
      const SearchListItem = Range.SEARCH_LIST_ITEMS.range;
      if (SearchListItem == null)
        throw new Error("range SearchListItem is not defined");
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
      const { container } = await renderWithConsole(<Harness />);
      expect(await screen.findByText(rng.name)).toBeTruthy();
      // createTestRange builds a range starting now, which formats as "Today ...".
      expect(container.textContent).toContain("Today");
    });
  });
});
