// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { screen } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/platform/layout";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const TYPE = "content-test-renderer";

const Renderer: Layout.Renderer = ({ layoutKey, visible, focused }) => (
  <div>
    <span data-testid="key">{layoutKey}</span>
    <span data-testid="visible">{String(visible)}</span>
    <span data-testid="focused">{String(focused)}</span>
  </div>
);
Renderer.displayName = "ContentTestRenderer";

const RENDERERS: Layout.Renderers = { [TYPE]: Renderer };

const place = (store: TestStore, key: string): void =>
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key,
        type: TYPE,
        name: key,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: key },
      }),
    );
  });

interface RenderContentArgs {
  forceHidden?: boolean;
}

const renderContent = async ({ forceHidden }: RenderContentArgs = {}) => {
  const store = await createTestStore();
  place(store, "l1");
  const result = await renderWithConsole(
    <Layout.RendererProvider value={RENDERERS}>
      <Layout.Content layoutKey="l1" forceHidden={forceHidden} />
    </Layout.RendererProvider>,
    { store },
  );
  return { ...result, store };
};

describe("layout Content", () => {
  it("renders the registered renderer with the layout key, visible by default", async () => {
    await renderContent();
    expect(screen.getByTestId("key").textContent).toBe("l1");
    expect(screen.getByTestId("visible").textContent).toBe("true");
    expect(screen.getByTestId("focused").textContent).toBe("false");
  });

  it("forces the renderer hidden when forceHidden is set", async () => {
    await renderContent({ forceHidden: true });
    expect(screen.getByTestId("visible").textContent).toBe("false");
  });

  it("marks the renderer focused and visible when its own layout is focused", async () => {
    const { store } = await renderContent();
    act(() => {
      store.dispatch(
        Session.Layout.setFocus({ windowKey: Drift.MAIN_WINDOW, key: "l1" }),
      );
    });
    expect(screen.getByTestId("focused").textContent).toBe("true");
    expect(screen.getByTestId("visible").textContent).toBe("true");
  });

  it("hides the renderer when a different layout is focused", async () => {
    const { store } = await renderContent();
    place(store, "l2");
    act(() => {
      store.dispatch(
        Session.Layout.setFocus({ windowKey: Drift.MAIN_WINDOW, key: "l2" }),
      );
    });
    expect(screen.getByTestId("visible").textContent).toBe("false");
    expect(screen.getByTestId("focused").textContent).toBe("false");
  });
});
