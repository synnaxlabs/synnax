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

import { Cluster } from "@/feature/cluster";
import { renderPalette } from "@/feature/command/testutil";
import { Session } from "@/session";
import { createCluster } from "@/session/cluster/testutil";

describe("Cluster Commands", () => {
  it("should deselect the active cluster when logging out", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Cluster.COMMANDS,
    });
    store.dispatch(Session.Cluster.set(createCluster("local", { name: "Local" })));
    store.dispatch(Session.Cluster.select("local"));
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("local");
    await openCommandPalette();
    await selectCommand("Log out");
    await waitFor(() =>
      expect(Session.Cluster.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
