// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, type resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, selectTestProject } from "@/testutil";

const client = createTestClient();

const createHarness = (item: resource.Resource) => {
  const SearchListItem = Range.SEARCH_LIST_ITEMS.range;
  if (SearchListItem == null) throw new Error("range SearchListItem is not defined");
  const Harness = (): ReactElement => {
    const staticProps = List.useStaticData<string, resource.Resource>({
      data: [item],
    });
    return (
      <Select.Frame<string, resource.Resource>
        {...staticProps}
        value={undefined}
        onChange={() => {}}
      >
        <SearchListItem key={item.key} itemKey={item.key} index={0} />
      </Select.Frame>
    );
  };
  return Harness;
};

describe("range/search", () => {
  describe("SearchListItem", () => {
    it("renders the resource name and time range", async () => {
      const rng = await createTestRange(client);
      const item = createResource(ranger.resourceID(rng.key), rng.name);
      const Harness = createHarness(item);
      const { wrapper } = await createConsoleWrapper({ client });
      const { container } = render(<Harness />, { wrapper });
      expect(await screen.findByText(rng.name)).toBeTruthy();
      // createTestRange builds a range starting now, which formats as "Today ...".
      await waitFor(() => expect(container.textContent).toContain("Today"));
    });

    it("adds the range to the session and opens it when selected", async () => {
      const rng = await createTestRange(client);
      const item = createResource(ranger.resourceID(rng.key), rng.name);
      const Harness = createHarness(item);
      const { wrapper, store } = await createConsoleWrapper({ client });
      await selectTestProject(store, client);
      const { container } = render(<Harness />, { wrapper });
      await waitFor(() => expect(container.textContent).toContain("Today"));
      fireEvent.click(await screen.findByText(rng.name), { detail: 0 });
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource).toEqual(ranger.resourceID(rng.key));
      expect(Session.Range.selectState(store.getState(), rng.key)).toEqual({
        variant: "persisted",
        key: rng.key,
      });
    });
  });
});
