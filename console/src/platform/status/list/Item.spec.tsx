// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type status } from "@synnaxlabs/client";
import { Component, List, Select } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Status } from "@/platform/status";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

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

const favoriteButton = (): HTMLButtonElement => {
  const btn = document.querySelector<HTMLButtonElement>(".console-favorite-button");
  if (btn == null) throw new Error("favorite button not found");
  return btn;
};

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
    await waitFor(() =>
      expect(favoriteButton().className).toContain("console--favorite"),
    );
  });

  it("should toggle the favorite in the slice when the favorite button is clicked", async () => {
    const stat = await createStatus();
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(<Fixture status={stat} />, { wrapper });
    await waitFor(() => expect(screen.getByText(stat.name)).toBeTruthy());
    expect(Session.Status.selectIsFavorite(store.getState(), stat.key)).toBe(false);
    fireEvent.click(favoriteButton());
    await waitFor(() =>
      expect(Session.Status.selectIsFavorite(store.getState(), stat.key)).toBe(true),
    );
  });
});
