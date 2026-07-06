// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Schematic } from "@/feature/schematic";
import { placeLayout } from "@/platform/layout/testutil";
import { Session } from "@/session";
import { createTestStore, renderWithConsole } from "@/testutil";

describe("Schematic.CONTEXT_MENUS", () => {
  it("renders the layout menu items and closes the schematic layout", async () => {
    const Menu = Schematic.CONTEXT_MENUS[Schematic.LAYOUT_TYPE];
    const store = await createTestStore();
    placeLayout(store, "s1");
    await renderWithConsole(<Menu layoutKey="s1" />, { store });
    fireEvent.click(await screen.findByText("Close"));
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), "s1")).toBeUndefined(),
    );
  });
});
