// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { List, Select } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Log } from "@/feature/log";
import { type Tree } from "@/platform/tree";
import { createEntry } from "@/platform/tree/testutil";
import { Session } from "@/session";
import { renderWithConsole, uniqueName } from "@/testutil";

describe("log/search", () => {
  it("places the log's layout when the search result is selected", async () => {
    const key = uuid.create();
    const name = uniqueName("log");
    const entry = createEntry(log.ontologyID(key), name);
    const SearchListItem = Log.SEARCH_LIST_ITEMS.log;
    if (SearchListItem == null) throw new Error("log SearchListItem is not defined");
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
    const { store } = await renderWithConsole(<Harness />);
    fireEvent.click(await screen.findByText(name), { detail: 0 });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), key);
      expect(placed?.type).toBe(Log.LAYOUT_TYPE);
      expect(placed?.name).toBe(name);
    });
  });
});
