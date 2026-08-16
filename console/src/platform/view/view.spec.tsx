// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Channel, Component, type List, Menu } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { View } from "@/platform/view";
import { enableEditing } from "@/platform/view/testutil";
import { createConsoleWrapper, getIconButton, uniqueName } from "@/testutil";

const RenderItem = ({ itemKey }: List.ItemProps<channel.Key>): ReactElement => (
  <span>{String(itemKey)}</span>
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

const client = createTestClient();

const renderHarness = async (): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client });
  render(<Harness />, { wrapper });
};

describe("View", () => {
  it("hides the search input until the view is made editable", async () => {
    await renderHarness();
    await waitFor(() => expect(screen.getByText("All Channels")).toBeTruthy());
    expect(screen.queryByPlaceholderText("Search channels...")).toBeNull();
  });

  it("renders the search input once editing is enabled", async () => {
    await renderHarness();
    await enableEditing();
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Search channels...")).toBeTruthy(),
    );
  });

  it("filters the listed items down to those matching the search term", async () => {
    const alpha = await client.channels.create({
      name: uniqueName("alpha"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const bravo = await client.channels.create({
      name: uniqueName("bravo"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    await renderHarness();
    await enableEditing();
    const input = await waitFor(() =>
      screen.getByPlaceholderText<HTMLInputElement>("Search channels..."),
    );
    fireEvent.change(input, { target: { value: alpha.name } });
    await waitFor(() => {
      expect(screen.getByText(String(alpha.key))).toBeTruthy();
      expect(screen.queryByText(String(bravo.key))).toBeNull();
    });
  });

  it("shows the empty state when the search matches no channels", async () => {
    await renderHarness();
    await enableEditing();
    const input = await waitFor(() =>
      screen.getByPlaceholderText<HTMLInputElement>("Search channels..."),
    );
    // Core search fuzzy-matches per token: dictionary-ish words ("no", "channel")
    // hit real channels, so the term must be pure gibberish.
    fireEvent.change(input, { target: { value: uniqueName("zzqjxvwq") } });
    await waitFor(() => expect(screen.getByText("No channels found.")).toBeTruthy());
  });

  it("reveals the filter menu contents when the filter trigger is opened", async () => {
    await renderHarness();
    await enableEditing();
    const trigger = await waitFor(() => getIconButton(document.body, "filter"));
    fireEvent.click(trigger);
    await waitFor(() => expect(screen.getByText("Filter Option")).toBeTruthy());
  });
});
