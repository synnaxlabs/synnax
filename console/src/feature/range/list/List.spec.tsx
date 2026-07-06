// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ranger } from "@synnaxlabs/client";
import { Ranger, state } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/feature/range";
import { searchAndClickNestedLabel } from "@/platform/label/testutil";
import { Modals } from "@/platform/modals";
import { createTestRange } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  getIconButton,
  getInputByNodePlaceholder,
  stubGeometry,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

stubGeometry();

const Harness = (): ReactElement => {
  const { data, getItem, subscribe, retrieve } = Ranger.useList({});
  return (
    <Range.List.List
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      retrieve={retrieve}
      enableSearch
      enableFilters
      enableAddButton
    />
  );
};

const renderList = async (): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(
    <>
      <Harness />
      <Modals.Stack />
    </>,
    { wrapper },
  );
};

describe("range/list/List", () => {
  it("narrows the list to ranges matching the search term", async () => {
    const target = await createTestRange(client);
    const other = await createTestRange(client);
    await renderList();
    const input = await waitFor(() =>
      getInputByNodePlaceholder(document.body, "Search Ranges..."),
    );
    fireEvent.change(input, { target: { value: target.name } });
    await waitFor(() => expect(screen.getByText(target.name)).toBeTruthy());
    expect(screen.queryByText(other.name)).toBeNull();
  });

  it("opens the create range modal from the add button", async () => {
    await renderList();
    await waitFor(() => getInputByNodePlaceholder(document.body, "Search Ranges..."));
    fireEvent.click(await waitFor(() => getIconButton(document.body, "add")));
    expect(await screen.findByText("Save Locally")).toBeTruthy();
  });
});

describe("range/list/SelectFilters", () => {
  it("adds the chosen label to the request and resets pagination", async () => {
    const label = await client.labels.create({
      name: uniqueName("filter_label"),
      color: "#E774D0",
    });
    const onRequestChange = vi.fn();
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <Range.List.SelectFilters
        request={{ offset: 25, limit: 25 }}
        onRequestChange={onRequestChange}
      />,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Filter"));
    await searchAndClickNestedLabel(label.name);
    await waitFor(() => expect(onRequestChange).toHaveBeenCalled());
    const [setter] = onRequestChange.mock.calls[0] as [
      state.SetArg<ranger.RetrieveRequest>,
    ];
    const next = state.executeSetter(setter, { offset: 25, limit: 25 });
    expect(next.hasLabels).toEqual([label.key]);
    expect(next.offset).toBe(0);
    expect(next.limit).toBe(0);
  });

  it("renders the applied label filters as chips", async () => {
    const label = await client.labels.create({
      name: uniqueName("chip_label"),
      color: "#E774D0",
    });
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <Range.List.Filters
        request={{ hasLabels: [label.key] }}
        onRequestChange={vi.fn()}
      />,
      { wrapper },
    );
    expect(await screen.findByText("Labels")).toBeTruthy();
    expect(await screen.findByText(label.name)).toBeTruthy();
  });
});
