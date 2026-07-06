// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Layout } from "@/platform/layout";

const Foo: Layout.Renderer = () => null;
Foo.displayName = "Foo";

const rendererWrapper = (renderers: Layout.Renderers) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Layout.RendererProvider value={renderers}>{children}</Layout.RendererProvider>
  );
  Wrapper.displayName = "RendererWrapper";
  return Wrapper;
};

const contextMenuWrapper = (menus: Layout.ContextMenus) => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Layout.ContextMenuProvider value={menus}>{children}</Layout.ContextMenuProvider>
  );
  Wrapper.displayName = "ContextMenuWrapper";
  return Wrapper;
};

describe("layout context", () => {
  describe("useRenderer", () => {
    it("throws when no renderer is registered for the type", () => {
      expect(() => renderHook(() => Layout.useRenderer("missing"))).toThrow(
        "no renderer for layout type missing",
      );
    });

    it("returns the renderer registered for the type", () => {
      const { result } = renderHook(() => Layout.useRenderer("foo"), {
        wrapper: rendererWrapper({ foo: Foo }),
      });
      expect(result.current).toBe(Foo);
    });
  });

  describe("useNameHook", () => {
    it("returns undefined when the renderer defines no useName", () => {
      const { result } = renderHook(() => Layout.useNameHook("foo"), {
        wrapper: rendererWrapper({ foo: Foo }),
      });
      expect(result.current).toBeUndefined();
    });

    it("returns undefined for an unregistered type", () => {
      const { result } = renderHook(() => Layout.useNameHook("absent"), {
        wrapper: rendererWrapper({ foo: Foo }),
      });
      expect(result.current).toBeUndefined();
    });

    it("returns the renderer's useName when present", () => {
      const useName: Layout.UseName = () => ({
        retrieve: () => {},
        onRename: () => {},
      });
      const Named: Layout.Renderer = () => null;
      Named.displayName = "Named";
      Named.useName = useName;
      const { result } = renderHook(() => Layout.useNameHook("named"), {
        wrapper: rendererWrapper({ named: Named }),
      });
      expect(result.current).toBe(useName);
    });
  });

  describe("useContextMenuRenderer", () => {
    const Menu: Layout.ContextMenuRenderer = () => null;
    Menu.displayName = "Menu";

    it("returns null when no menu is registered for the type", () => {
      const { result } = renderHook(() => Layout.useContextMenuRenderer("absent"), {
        wrapper: contextMenuWrapper({ foo: Menu }),
      });
      expect(result.current).toBeNull();
    });

    it("returns null when there is no provider", () => {
      const { result } = renderHook(() => Layout.useContextMenuRenderer("foo"));
      expect(result.current).toBeNull();
    });

    it("returns the menu renderer registered for the type", () => {
      const { result } = renderHook(() => Layout.useContextMenuRenderer("foo"), {
        wrapper: contextMenuWrapper({ foo: Menu }),
      });
      expect(result.current).toBe(Menu);
    });
  });
});
