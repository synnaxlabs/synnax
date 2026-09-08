// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({
  engine: "web",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/plugin-shell", () => ({
  open: vi.fn().mockResolvedValue(undefined),
}));

import { open } from "@tauri-apps/plugin-shell";

import { Runtime } from "@/platform/runtime";

const openMock = vi.mocked(open);

const dispatchClick = (el: Element, type: "click" | "auxclick" = "click") =>
  el.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));

const createAnchor = (attrs: Record<string, string>): HTMLAnchorElement => {
  const a = document.createElement("a");
  for (const [k, v] of Object.entries(attrs)) a.setAttribute(k, v);
  document.body.appendChild(a);
  return a;
};

describe("Runtime.useExternalLinkHandler", () => {
  beforeEach(() => {
    mocks.engine = "web";
    openMock.mockClear();
    openMock.mockResolvedValue(undefined);
    document.body.innerHTML = "";
  });
  afterEach(() => {
    document.body.innerHTML = "";
  });

  describe("web engine", () => {
    it("should not intercept target=_blank clicks", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      dispatchClick(a);
      expect(openMock).not.toHaveBeenCalled();
    });
  });

  describe("tauri engine", () => {
    beforeEach(() => {
      mocks.engine = "tauri";
    });

    it("should open target=_blank links via the shell plugin", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      const event = new MouseEvent("click", { bubbles: true, cancelable: true });
      a.dispatchEvent(event);
      expect(openMock).toHaveBeenCalledTimes(1);
      expect(openMock).toHaveBeenCalledWith(a.href);
      expect(event.defaultPrevented).toBe(true);
    });

    it("should route the click through the resolved absolute href", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "/relative/path" });
      dispatchClick(a);
      expect(openMock).toHaveBeenCalledWith(a.href);
      expect(a.href).toContain("/relative/path");
    });

    it("should handle clicks on descendants of the anchor", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      const span = document.createElement("span");
      a.appendChild(span);
      dispatchClick(span);
      expect(openMock).toHaveBeenCalledWith(a.href);
    });

    it("should intercept auxclick events", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      dispatchClick(a, "auxclick");
      expect(openMock).toHaveBeenCalledWith(a.href);
    });

    it("should ignore anchors without target=_blank", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ href: "https://example.com" });
      dispatchClick(a);
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should ignore anchors with an empty href", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank" });
      dispatchClick(a);
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should ignore clicks that are not on an anchor", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      const div = document.createElement("div");
      document.body.appendChild(div);
      dispatchClick(div);
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should ignore events whose target is not an element", () => {
      renderHook(() => Runtime.useExternalLinkHandler());
      document.dispatchEvent(
        new MouseEvent("click", { bubbles: true, cancelable: true }),
      );
      expect(openMock).not.toHaveBeenCalled();
    });

    it("should log an error when the shell plugin rejects", async () => {
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      openMock.mockRejectedValueOnce(new Error("no shell"));
      renderHook(() => Runtime.useExternalLinkHandler());
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      dispatchClick(a);
      await vi.waitFor(() => expect(errorSpy).toHaveBeenCalled());
      expect(errorSpy.mock.calls[0][0]).toContain(a.href);
      errorSpy.mockRestore();
    });

    it("should stop intercepting after unmount", () => {
      const { unmount } = renderHook(() => Runtime.useExternalLinkHandler());
      unmount();
      const a = createAnchor({ target: "_blank", href: "https://example.com" });
      dispatchClick(a);
      expect(openMock).not.toHaveBeenCalled();
    });
  });
});
