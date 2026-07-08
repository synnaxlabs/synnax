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

import { Layout } from "@/platform/layout";
import { renderWithConsole } from "@/testutil";

// The default store seeds a "main" layout (type "main") in the main window; register a
// renderer for it so the window mounts its content.
const MainRenderer: Layout.Renderer = ({ layoutKey }) => (
  <div>{`main content:${layoutKey}`}</div>
);
MainRenderer.displayName = "MainRenderer";

describe("layout Window", () => {
  it("renders the main window's layout through the registered renderer", async () => {
    await renderWithConsole(
      <Layout.RendererProvider value={{ main: MainRenderer }}>
        <Layout.Window />
      </Layout.RendererProvider>,
    );
    await waitFor(() => expect(screen.getByText("main content:main")).toBeTruthy());
  });

  it("opens the default context menu on right click", async () => {
    await renderWithConsole(
      <Layout.RendererProvider value={{ main: MainRenderer }}>
        <Layout.Window />
      </Layout.RendererProvider>,
    );
    const box = await waitFor(() => {
      const el = document.body.querySelector(".console-main");
      if (el == null) throw new Error("main window box not found");
      return el;
    });
    fireEvent.contextMenu(box);
    await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
  });
});
