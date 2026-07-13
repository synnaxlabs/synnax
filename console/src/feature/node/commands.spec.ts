// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Node } from "@/feature/node";
import { Session } from "@/session";
import { stubGeometry } from "@/testutil";

stubGeometry();

describe("Cluster Commands", () => {
  it("should deselect the active cluster when logging out", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Node.COMMANDS,
    });
    store.dispatch(
      Session.Node.set({
        key: "local",
        name: "Local",
        host: "localhost",
        port: 9090,
        username: "synnax",
        password: "seldon",
        secure: false,
      }),
    );
    store.dispatch(Session.Node.select("local"));
    expect(Session.Node.selectSelectedKey(store.getState())).toBe("local");
    await openCommandPalette();
    await selectCommand("Log out");
    await waitFor(() =>
      expect(Session.Node.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
