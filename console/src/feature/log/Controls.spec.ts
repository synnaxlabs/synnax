// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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

  it("should let a viewer search", async () => {
    const { key, store } = await renderLog(Controls, { as: await roles.get("Viewer") });
    await openSearch("err");
    await waitFor(() => expect(searchIn(store, key)?.text).toBe("err"));
  });
});
