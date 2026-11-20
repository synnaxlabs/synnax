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
import { Status } from "@synnaxlabs/lyra/status";
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { createTestRange } from "@/platform/range/testutil";
import { createResource } from "@/platform/tree/testutil";
import {
  createConsoleWrapper,
  resolveFocusedTab,
  selectRange,
  selectTestProject,
} from "@/testutil";

const client = createTestClient();

/** Creates a range that starts now, so its start formats as "Today ...". */
const createTodayRange = async (): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await createTestRange(client, {
    timeRange: new TimeRange(start, start.add(TimeSpan.HOUR)),
  });
};

const createHarness = (
  item: resource.Resource,
  onStatuses: (statuses: Status.NotificationSpec[]) => void = () => {},
) => {
  const SearchListItem = Range.SEARCH_LIST_ITEMS.range;
  if (SearchListItem == null) throw new Error("range SearchListItem is not defined");
  const Harness = (): ReactElement => {
    onStatuses(Status.useNotifications().statuses);
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
      const rng = await createTodayRange();
      const item = createResource(ranger.resourceID(rng.key), rng.name);
      const Harness = createHarness(item);
      const { wrapper } = await createConsoleWrapper({ client });
      const { container } = render(<Harness />, { wrapper });
      expect(await screen.findByText(rng.name)).toBeTruthy();
      await waitFor(() => expect(container.textContent).toContain("Today"));
    });

    it("opens the range without adding it to the favorites", async () => {
      const rng = await createTodayRange();
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
      expect(selectRange(store.getState(), rng.key)).toBeUndefined();
    });

    it("opens the range when selected before it loads", async () => {
      const rng = await createTestRange(client);
      // A client opened after the create does not hold the range, so it loads late.
      const fresh = createTestClient();
      const item = createResource(ranger.resourceID(rng.key), rng.name);
      const Harness = createHarness(item);
      const { wrapper, store } = await createConsoleWrapper({ client: fresh });
      await selectTestProject(store, fresh);
      render(<Harness />, { wrapper });
      fireEvent.click(await screen.findByText(rng.name), { detail: 0 });
      const tab = await resolveFocusedTab(store, fresh);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource).toEqual(ranger.resourceID(rng.key));
    });

    it("reports an error when the selected range does not exist", async () => {
      const item = createResource(ranger.resourceID(uuid.create()), "missing");
      let statuses: Status.NotificationSpec[] = [];
      const Harness = createHarness(item, (s) => (statuses = s));
      const { wrapper } = await createConsoleWrapper({ client });
      render(<Harness />, { wrapper });
      fireEvent.click(await screen.findByText("missing"), { detail: 0 });
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            variant: "error",
            name: "missing",
            message: "Failed to select range",
          }),
        ),
      );
    });
  });
});
