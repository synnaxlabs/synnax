// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RoleClients } from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Controls } from "@/feature/log/Controls";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";

const roles = new RoleClients(client);

type Store = Awaited<ReturnType<typeof renderLog>>["store"];

const searchIn = (store: Store, key: string): Session.Log.SearchState | undefined =>
  documentIn(store.getState()[Session.Log.SLICE_NAME], key)?.search;

const openSearch = async (text: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Search" }));
  fireEvent.change(screen.getByLabelText("Search text"), { target: { value: text } });
};

describe("log/Controls search", () => {
  it("should store the search text, mode, and filter", async () => {
    const { key, store } = await renderLog(Controls);
    await openSearch("err");
    fireEvent.click(screen.getByRole("button", { name: "Use regex" }));
    fireEvent.click(screen.getByRole("button", { name: "Show only matching lines" }));
    await waitFor(() =>
      expect(searchIn(store, key)).toEqual({
        text: "err",
        mode: "regex",
        filtered: true,
      }),
    );
  });

  it("should show an error only for an invalid regex", async () => {
    await renderLog(Controls);
    await openSearch("(");
    expect(screen.queryByText("Invalid regex")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Use regex" }));
    expect(await screen.findByText("Invalid regex")).toBeDefined();
  });

  it("should end the search when the close button is clicked", async () => {
    const { key, store } = await renderLog(Controls);
    await openSearch("err");
    fireEvent.click(screen.getByRole("button", { name: "Close search" }));
    await waitFor(() => expect(searchIn(store, key)?.text).toBe(""));
    expect(screen.queryByLabelText("Search text")).toBeNull();
  });

  it("should end the search on Escape", async () => {
    const { key, store } = await renderLog(Controls);
    await openSearch("err");
    fireEvent.keyDown(screen.getByLabelText("Search text"), { key: "Escape" });
    await waitFor(() => expect(searchIn(store, key)?.text).toBe(""));
    expect(screen.queryByLabelText("Search text")).toBeNull();
  });

  it("should show a stored search when the controls mount", async () => {
    await renderLog(Controls, {
      preloadedState: (key) => ({
        [Session.Log.SLICE_NAME]: {
          ...Session.Log.ZERO_SLICE_STATE,
          windows: {
            [MAIN_WINDOW]: {
              [key]: {
                ...Session.Log.ZERO_STATE,
                search: { ...Session.Log.ZERO_STATE.search, text: "err" },
              },
            },
          },
        },
      }),
    });
    expect(await screen.findByDisplayValue("err")).toBeDefined();
  });

  it("should keep a copy in the search box from reaching the log", async () => {
    const onCopy = vi.fn();
    const InLog = (): ReactElement =>
      createElement("div", { onCopy }, createElement(Controls));
    await renderLog(InLog);
    await openSearch("err");
    fireEvent.copy(screen.getByLabelText("Search text"));
    expect(onCopy).not.toHaveBeenCalled();
  });

  it("should let a viewer search", async () => {
    const { key, store } = await renderLog(Controls, { as: await roles.get("Viewer") });
    await openSearch("err");
    await waitFor(() => expect(searchIn(store, key)?.text).toBe("err"));
  });
});
