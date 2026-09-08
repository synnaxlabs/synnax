// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Component, List, Select, Text } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Status } from "@/platform/status";
import { Session } from "@/session";
import {
  awaitTextEditing,
  commitTextEdit,
  countEditableText,
  createConsoleWrapper,
  uniqueName,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const createStatus = async (message = "a status message"): Promise<status.Status> =>
  await client.statuses.set({
    name: uniqueName("status"),
    message,
    variant: "warning",
  });

const item = Component.renderProp(Status.List.Item);

const Fixture = ({ status: stat }: { status: status.Status }): ReactElement => {
  const { data, getItem } = List.useStaticData<status.Key, status.Status>({
    data: [stat],
  });
  return (
    <Select.Frame<status.Key, status.Status>
      multiple
      data={data}
      getItem={getItem}
      value={[]}
      onChange={vi.fn()}
    >
      <List.Items>{item}</List.Items>
    </Select.Frame>
  );
};
Fixture.displayName = "Fixture";

const favoriteCheckbox = (): HTMLInputElement =>
  screen.getByRole("checkbox", { name: "Favorite" });

describe("Status.List.Item", () => {
  it("should render the status name and message", async () => {
    const stat = await createStatus("something happened");
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture status={stat} />, { wrapper });
    await waitFor(() => expect(screen.getByText(stat.name)).toBeTruthy());
    expect(screen.getByText("something happened")).toBeTruthy();
  });

  it("should reflect the favorite state from the slice", async () => {
    const stat = await createStatus();
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: {
        [Session.Status.SLICE_NAME]: { version: 0, favorites: [stat.key] },
      },
    });
    render(<Fixture status={stat} />, { wrapper });
    await waitFor(() => expect(favoriteCheckbox().checked).toBe(true));
  });

  it("should toggle the favorite in the slice when the favorite button is clicked", async () => {
    const stat = await createStatus();
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Fixture status={stat} />, { wrapper });
    await waitFor(() => expect(screen.getByText(stat.name)).toBeTruthy());
    expect(Session.Status.selectIsFavorite(store.getState(), stat.key)).toBe(false);
    fireEvent.click(favoriteCheckbox());
    await waitFor(() =>
      expect(Session.Status.selectIsFavorite(store.getState(), stat.key)).toBe(true),
    );
  });

  it("should rename the status in place", async () => {
    const stat = await createStatus();
    const { wrapper } = await createConsoleWrapper({ client });
    render(<Fixture status={stat} />, { wrapper });
    await screen.findByText(stat.name);
    await waitFor(() => expect(countEditableText(List.itemNameID(stat.key))).toBe(1));
    Text.edit(List.itemNameID(stat.key));
    const editor = await awaitTextEditing(List.itemNameID(stat.key));
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.statuses.retrieve(stat.key)).name).toBe(renamed),
    );
  });

  it("should render the name as plain text for a viewer", async () => {
    const stat = await createStatus();
    const { wrapper } = await createConsoleWrapper({
      client: await roles.get("Viewer"),
    });
    render(<Fixture status={stat} />, { wrapper });
    await screen.findByText(stat.name);
    await act(async () => {});
    expect(countEditableText(List.itemNameID(stat.key))).toBe(0);
  });
});
