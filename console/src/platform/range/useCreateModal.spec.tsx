// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { Range } from "@/platform/range";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const Harness = ({ params }: { params?: Range.CreateModalParams }): ReactElement => {
  const open = Range.useCreateModal();
  return <button onClick={() => open(params)}>open</button>;
};
Harness.displayName = "Harness";

const openModal = async (params?: Range.CreateModalParams) => {
  const rendered = await renderWithConsole(
    <>
      <Harness params={params} />
      <Modals.Stack />
    </>,
  );
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  await waitFor(() => expect(screen.getByText("Save Locally")).toBeTruthy());
  return rendered;
};

const clickButtonWithText = (text: string): void => {
  const btn = screen
    .getAllByText(text)
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error(`button with text ${text} not found`);
  fireEvent.click(btn);
};

describe("Range.useCreateModal", () => {
  it("should render the create form with save actions", async () => {
    await openModal();
    expect(screen.getByText("Save Locally")).toBeTruthy();
    expect(screen.getAllByText("Save to Synnax").length).toBeGreaterThan(0);
    expect(screen.getByText("Stage")).toBeTruthy();
  });

  it("should disable Save to Synnax when there is no connected client", async () => {
    await openModal();
    const btn = screen
      .getAllByText("Save to Synnax")
      .map((el) => el.closest<HTMLButtonElement>("button"))
      .find((b) => b != null);
    expect(btn?.className).toContain("pluto--disabled");
  });

  it("should add a non-persisted range to the slice and close on Save Locally", async () => {
    const { store } = await openModal({
      name: "Local Range",
      timeRange: { start: 1000, end: 2000 },
    });
    clickButtonWithText("Save Locally");
    await waitFor(() => expect(screen.queryByText("Save Locally")).toBeNull());
    const ranges = Session.Range.selectMultiple(store.getState());
    const created = ranges.find((r) => r.name === "Local Range");
    expect(created).toBeDefined();
    expect(created?.persisted).toBe(false);
    expect(created?.variant).toBe("static");
  });

  it("should not save locally when the name is empty", async () => {
    const { store } = await openModal({ timeRange: { start: 1000, end: 2000 } });
    const before = Session.Range.selectMultiple(store.getState()).length;
    clickButtonWithText("Save Locally");
    expect(screen.getByText("Save Locally")).toBeTruthy();
    expect(Session.Range.selectMultiple(store.getState()).length).toBe(before);
  });

  it("should persist the range to the cluster and favorite it on Save to Synnax", async () => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    render(
      <>
        <Harness
          params={{ name: "Persisted Range", timeRange: { start: 1, end: 2 } }}
        />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(() => expect(screen.getByText("Save Locally")).toBeTruthy());
    clickButtonWithText("Save to Synnax");
    await waitFor(() => {
      const created = Session.Range.selectMultiple(store.getState()).find(
        (r) => r.name === "Persisted Range",
      );
      expect(created).toBeDefined();
      expect(created?.persisted).toBe(true);
    }, TIMEOUT);
    await waitFor(() => expect(screen.queryByText("Save Locally")).toBeNull());
  });
});
