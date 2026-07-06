// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { createPreloadedState } from "@/feature/lineplot/testutil";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("lineplot ContextMenu", () => {
  it("closes the plot's mosaic tab through the layout menu", async () => {
    const key = uuid.create();
    const Menu = LinePlot.CONTEXT_MENUS[LinePlot.LAYOUT_TYPE];
    const { store } = await renderWithConsole(<Menu layoutKey={key} />, {
      preloadedState: createPreloadedState(key, "Test Plot"),
    });
    fireEvent.click(await screen.findByText("Close"));
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), key)).toBeUndefined(),
    );
  });
});
