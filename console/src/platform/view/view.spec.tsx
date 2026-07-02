// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, createTestClient } from "@synnaxlabs/client";
import { Channel, Component, type List, Menu } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { View } from "@/platform/view";
import { createConsoleWrapper } from "@/testutil";

const RenderItem = ({ itemKey }: List.ItemProps<channel.Key>): ReactElement => (
  <span data-testid="view-item">{String(itemKey)}</span>
);
const item = Component.renderProp(RenderItem);

const Inner = (): ReactElement => {
  const listProps = Channel.useList({
    initialQuery: View.useContext().getInitialView().query,
  });
  return (
    <View.Form {...listProps}>
      <View.Toolbar>
        <View.FilterMenu>
          <Menu.Item itemKey="only">Filter Option</Menu.Item>
        </View.FilterMenu>
        <View.Search />
      </View.Toolbar>
      <View.Items<channel.Key>>{item}</View.Items>
    </View.Form>
  );
};

const Harness = (): ReactElement => (
  <View.Frame resourceType="channel" icon="Channel">
    <Inner />
  </View.Frame>
);
Harness.displayName = "Harness";

const TIMEOUT = { timeout: 5000 };

const renderHarness = async (): Promise<void> => {
  const client = createTestClient();
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness />, { wrapper });
};

// The view begins non-editable until the async update-permission check resolves; the
// toolbar (search, filter menu) only mounts once editing is enabled via the toggle.
const enableEditing = async (): Promise<void> => {
  const toggle = await waitFor(() => {
    const btn = screen.getByLabelText("pluto-icon--edit").closest("button");
    if (btn == null) throw new Error("edit toggle not found");
    return btn;
  }, TIMEOUT);
  fireEvent.click(toggle);
};

describe("View", () => {
  it("renders the static 'all resources' view for the resource type", async () => {
    await renderHarness();
    await waitFor(() => expect(screen.getByText("All Channels")).toBeTruthy(), TIMEOUT);
  });

  it("hides the search input until the view is made editable", async () => {
    await renderHarness();
    await waitFor(() => expect(screen.getByText("All Channels")).toBeTruthy(), TIMEOUT);
    expect(screen.queryByPlaceholderText("Search channels...")).toBeNull();
  });

  it("renders the search input once editing is enabled", async () => {
    await renderHarness();
    await enableEditing();
    await waitFor(
      () => expect(screen.getByPlaceholderText("Search channels...")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("updates the search input value as the user types", async () => {
    await renderHarness();
    await enableEditing();
    const input = await waitFor(
      () => screen.getByPlaceholderText<HTMLInputElement>("Search channels..."),
      TIMEOUT,
    );
    fireEvent.change(input, { target: { value: "sensor" } });
    expect(input.value).toBe("sensor");
  });

  it("reveals the filter menu contents when the filter trigger is opened", async () => {
    await renderHarness();
    await enableEditing();
    const trigger = await waitFor(() => {
      const btn = screen.getByLabelText("pluto-icon--filter").closest("button");
      if (btn == null) throw new Error("filter trigger not found");
      return btn;
    }, TIMEOUT);
    fireEvent.click(trigger);
    await waitFor(
      () => expect(screen.getByText("Filter Option")).toBeTruthy(),
      TIMEOUT,
    );
  });
});
