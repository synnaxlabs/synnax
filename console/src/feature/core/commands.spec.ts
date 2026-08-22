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

import { Core } from "@/feature/core";
import { renderPalette } from "@/feature/command/testutil";
import { Session } from "@/session";
import { createCore } from "@/session/core/testutil";

describe("Core Commands", () => {
  it("should deselect the active Core when logging out", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Core.COMMANDS,
    });
    store.dispatch(Session.Core.set(createCore("local", { name: "Local" })));
    store.dispatch(Session.Core.select("local"));
    expect(Session.Core.selectSelectedKey(store.getState())).toBe("local");
    await openCommandPalette();
    await selectCommand("Log out");
    await waitFor(() =>
      expect(Session.Core.selectSelectedKey(store.getState())).toBeUndefined(),
    );
  });
});
