// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { act, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/platform/layout";
import { placeLayout } from "@/platform/layout/testutil";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const TYPE = "content-test-renderer";

const Renderer: Layout.Renderer = ({ layoutKey, visible, focused, onClose }) => (
  <div>
    <span>{`key:${layoutKey}`}</span>
    <span>{`visible:${String(visible)}`}</span>
    <span>{`focused:${String(focused)}`}</span>
    <button onClick={onClose}>close from renderer</button>
  </div>
);
Renderer.displayName = "ContentTestRenderer";

const RENDERERS: Layout.Renderers = { [TYPE]: Renderer };

const place = (store: TestStore, key: string): void =>
  placeLayout(store, key, { type: TYPE });

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
    expect(screen.getByText("key:l1")).toBeTruthy();
    expect(screen.getByText("visible:true")).toBeTruthy();
    expect(screen.getByText("focused:false")).toBeTruthy();
  });

  it("forces the renderer hidden when forceHidden is set", async () => {
    await renderContent({ forceHidden: true });
    expect(screen.getByText("visible:false")).toBeTruthy();
  });

  it("marks the renderer focused and visible when its own layout is focused", async () => {
    const { store } = await renderContent();
    act(() => {
      store.dispatch(
        Session.Layout.setFocus({ windowKey: Drift.MAIN_WINDOW, key: "l1" }),
      );
    });
    expect(screen.getByText("focused:true")).toBeTruthy();
    expect(screen.getByText("visible:true")).toBeTruthy();
  });

  it("hides the renderer when a different layout is focused", async () => {
    const { store } = await renderContent();
    place(store, "l2");
    act(() => {
      store.dispatch(
        Session.Layout.setFocus({ windowKey: Drift.MAIN_WINDOW, key: "l2" }),
      );
    });
    expect(screen.getByText("visible:false")).toBeTruthy();
    expect(screen.getByText("focused:false")).toBeTruthy();
  });

  it("removes the layout from the store when the renderer invokes onClose", async () => {
    const Host = (): ReactElement => {
      const exists = Session.Layout.useSelect("l1") != null;
      if (!exists) return <span>layout gone</span>;
      return <Layout.Content layoutKey="l1" />;
    };
    const store = await createTestStore();
    place(store, "l1");
    const result = await renderWithConsole(
      <Layout.RendererProvider value={RENDERERS}>
        <Host />
      </Layout.RendererProvider>,
      { store },
    );
    expect(Session.Layout.select(store.getState(), "l1")).toBeDefined();
    fireEvent.click(screen.getByText("close from renderer"));
    await waitFor(() =>
      expect(Session.Layout.select(result.store.getState(), "l1")).toBeUndefined(),
    );
    expect(screen.getByText("layout gone")).toBeTruthy();
  });
});
