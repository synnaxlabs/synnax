// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Channel } from "@/feature/channel";
import { Search } from "@/feature/search";
import { Search as PlatformSearch } from "@/platform/search";
import { createConsoleWrapper, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

const PLACEHOLDER = <>Search</>;

const Harness = ({
  items = Channel.SEARCH_LIST_ITEMS,
}: {
  items?: PlatformSearch.ListItems;
}): ReactElement => {
  const [value, setValue] = useState("");
  return (
    <Search.List
      items={items}
      value={value}
      inputPlaceholder={PLACEHOLDER}
      onChange={setValue}
    />
  );
};

const renderSearch = async (items?: PlatformSearch.ListItems): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness items={items} />, { wrapper });
};

const searchInput = (): HTMLInputElement => {
  const input = document.querySelector<HTMLInputElement>(
    ".console-palette__input input",
  );
  if (input == null) throw new Error("search input not found");
  return input;
};

describe("Search.List", () => {
  it("shows the empty state when no resource matches the query", async () => {
    await renderSearch();
    fireEvent.change(searchInput(), { target: { value: uniqueName("no_such") } });
    await waitFor(() => expect(screen.getByText("No resources found")).toBeTruthy());
  });

  it("finds a live resource by name and renders its search item", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    await renderSearch();
    fireEvent.change(searchInput(), { target: { value: ch.name } });
    await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
  });

  it("fires a search item's onSelect exactly once per mouse click", async () => {
    const ch = await client.channels.create({
      name: uniqueName("ch"),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const onSelect = vi.fn();
    const TestItem = PlatformSearch.createListItem({
      icon: <Icon.Channel />,
      useOnSelect: () => onSelect,
    });
    await renderSearch({ channel: TestItem });
    fireEvent.change(searchInput(), { target: { value: ch.name } });
    await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy());
    // A real mouse click carries detail 1. Selection re-dispatches a synthetic
    // click (detail 0) on the item, which is the only click allowed to fire
    // onSelect; the item must not also fire it for the original click.
    fireEvent.click(screen.getByText(ch.name), { detail: 1 });
    await waitFor(() => expect(onSelect).toHaveBeenCalledTimes(1));
    expect(onSelect.mock.calls[0][0].name).toEqual(ch.name);
  });
});
