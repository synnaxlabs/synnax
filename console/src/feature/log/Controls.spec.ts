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
import { aether } from "@synnaxlabs/pluto/ether";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { createElement, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Log } from "@/feature/log/Log";
import { client, renderLog } from "@/feature/log/toolbar/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";

const roles = new RoleClients(client);

const stateZ = z.looseObject({
  jump: z.object({ request: z.number() }).optional(),
});

// The real worker needs a canvas. This one finds no match for any jump.
class WorkerStub extends aether.Leaf<typeof stateZ> {
  static readonly TYPE = "log";
  schema = stateZ;

  afterUpdate(): void {
    if ((this.state.jump?.request ?? 0) > 0 && this.state.jumpStatus !== "none")
      this.setState((s) => ({ ...s, jumpStatus: "none" }));
  }
}

const REGISTRY = { [WorkerStub.TYPE]: WorkerStub };

type Store = Awaited<ReturnType<typeof renderLog>>["store"];

const searchIn = (store: Store, key: string): Session.Log.SearchState | undefined =>
  documentIn(store.getState()[Session.Log.SLICE_NAME], key)?.search;

const openSearch = async (text: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Search" }));
  fireEvent.change(screen.getByLabelText("Search text"), { target: { value: text } });
};

describe("log/Controls search", () => {
  it("should store the search text, mode, and filter", async () => {
    const { key, store } = await renderLog(Log, { additionalRegistry: REGISTRY });
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
    await renderLog(Log, { additionalRegistry: REGISTRY });
    await openSearch("(");
    expect(screen.queryByText("Invalid regex")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Use regex" }));
    expect(await screen.findByText("Invalid regex")).toBeDefined();
  });

  it("should pause the log on a jump and show when no match is left", async () => {
    const { key, store } = await renderLog(Log, { additionalRegistry: REGISTRY });
    await openSearch("err");
    fireEvent.keyDown(screen.getByLabelText("Search text"), { key: "Enter" });
    expect(await screen.findByText("No more matches")).toBeDefined();
    expect(documentIn(store.getState()[Session.Log.SLICE_NAME], key)?.hold).toBe(true);
  });

  it("should jump from the previous and next buttons", async () => {
    await renderLog(Log, { additionalRegistry: REGISTRY });
    await openSearch("err");
    fireEvent.click(screen.getByRole("button", { name: "Previous match" }));
    expect(await screen.findByText("No more matches")).toBeDefined();
  });

  it("should end the search when the close button is clicked", async () => {
    const { key, store } = await renderLog(Log, { additionalRegistry: REGISTRY });
    await openSearch("err");
    fireEvent.click(screen.getByRole("button", { name: "Close search" }));
    await waitFor(() => expect(searchIn(store, key)?.text).toBe(""));
    expect(screen.queryByLabelText("Search text")).toBeNull();
  });

  it("should end the search on Escape", async () => {
    const { key, store } = await renderLog(Log, { additionalRegistry: REGISTRY });
    await openSearch("err");
    fireEvent.keyDown(screen.getByLabelText("Search text"), { key: "Escape" });
    await waitFor(() => expect(searchIn(store, key)?.text).toBe(""));
    expect(screen.queryByLabelText("Search text")).toBeNull();
  });

  it("should show a stored search when the controls mount", async () => {
    await renderLog(Log, {
      additionalRegistry: REGISTRY,
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
    const InPage = (): ReactElement =>
      createElement("div", { onCopy }, createElement(Log));
    await renderLog(InPage, { additionalRegistry: REGISTRY });
    await openSearch("err");
    fireEvent.copy(screen.getByLabelText("Search text"));
    expect(onCopy).not.toHaveBeenCalled();
  });

  it("should let a viewer search", async () => {
    const { key, store } = await renderLog(Log, {
      additionalRegistry: REGISTRY,
      as: await roles.get("Viewer"),
    });
    await openSearch("err");
    await waitFor(() => expect(searchIn(store, key)?.text).toBe("err"));
  });
});
