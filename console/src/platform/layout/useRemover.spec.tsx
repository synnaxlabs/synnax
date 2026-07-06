// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/platform/layout";
import { placeLayout } from "@/platform/layout/testutil";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import { createTestStore, renderWithConsole } from "@/testutil";

const Harness = ({ layoutKey }: { layoutKey: string }): ReactElement => {
  const remove = Layout.useRemover();
  return (
    <>
      <button onClick={() => remove(layoutKey)}>remove</button>
      <Modals.Stack />
    </>
  );
};
Harness.displayName = "Harness";

const setup = async (key: string) => {
  const store = await createTestStore();
  placeLayout(store, key, { type: "remover-test", unsavedChanges: true });
  const result = await renderWithConsole(<Harness layoutKey={key} />, { store });
  return { ...result, store };
};

describe("useRemover with unsaved changes", () => {
  it("prompts for confirmation before removing an unsaved layout", async () => {
    await setup("l1");
    fireEvent.click(screen.getByText("remove"));
    await waitFor(() =>
      expect(screen.getByText(/l1 has unsaved changes/)).toBeTruthy(),
    );
  });

  it("removes the layout when the user confirms", async () => {
    const { store } = await setup("l1");
    fireEvent.click(screen.getByText("remove"));
    await waitFor(() =>
      expect(screen.getByText(/l1 has unsaved changes/)).toBeTruthy(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), "l1")).toBeUndefined(),
    );
  });

  it("keeps the layout when the user cancels", async () => {
    const { store } = await setup("l1");
    fireEvent.click(screen.getByText("remove"));
    await waitFor(() =>
      expect(screen.getByText(/l1 has unsaved changes/)).toBeTruthy(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() =>
      expect(screen.queryByText(/l1 has unsaved changes/)).toBeNull(),
    );
    expect(Session.Layout.select(store.getState(), "l1")).toBeDefined();
  });
});
