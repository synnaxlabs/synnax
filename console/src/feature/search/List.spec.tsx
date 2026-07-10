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
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Search } from "@/feature/search";
import { createConsoleWrapper, stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

const PLACEHOLDER = <>Search</>;

const Harness = (): ReactElement => {
  const [value, setValue] = useState("");
  return (
    <Search.List
      items={Channel.SEARCH_LIST_ITEMS}
      value={value}
      inputPlaceholder={PLACEHOLDER}
      onChange={setValue}
    />
  );
};

const renderSearch = async (): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness />, { wrapper });
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
});
