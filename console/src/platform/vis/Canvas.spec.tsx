// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { runtime } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

// The handoff is chosen once at load, so each run needs fresh modules.
const renderCanvas = async (engine: "web" | "tauri", os: runtime.OS) => {
  mocks.engine = engine;
  vi.resetModules();
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS(os);
  const { canvas } = await import("@synnaxlabs/pluto/ether");
  const { Vis } = await import("@/platform/vis");
  const { createConsoleWrapper } = await import("@/testutil");
  const { wrapper } = await createConsoleWrapper({
    client: null,
    additionalRegistry: canvas.REGISTRY,
  });
  render(<Vis.Canvas />, { wrapper });
};

describe("Vis.Canvas", () => {
  const transfer = vi.fn(() => ({ getContext: () => null }));
  const getContext = vi.fn(() => ({ transferFromImageBitmap: () => {} }));
  const baseGetContext = Object.getOwnPropertyDescriptor(
    HTMLCanvasElement.prototype,
    "getContext",
  );

  beforeEach(() => {
    transfer.mockClear();
    getContext.mockClear();
    // jsdom has no OffscreenCanvas, so the worker gets no WebGL context.
    Object.defineProperty(globalThis, "OffscreenCanvas", {
      configurable: true,
      value: class {
        getContext = () => null;
      },
    });
    Object.defineProperty(HTMLCanvasElement.prototype, "transferControlToOffscreen", {
      configurable: true,
      value: transfer,
    });
    Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
      configurable: true,
      value: getContext,
    });
  });

  afterEach(() => {
    mocks.engine = "web";
    Reflect.deleteProperty(globalThis, "OffscreenCanvas");
    Reflect.deleteProperty(HTMLCanvasElement.prototype, "transferControlToOffscreen");
    if (baseGetContext != null)
      Object.defineProperty(HTMLCanvasElement.prototype, "getContext", baseGetContext);
  });

  it("should hand frames off as bitmaps in the desktop app on Linux", async () => {
    await renderCanvas("tauri", "Linux");
    await waitFor(() => expect(getContext).toHaveBeenCalledWith("bitmaprenderer"));
    expect(transfer).not.toHaveBeenCalled();
  });

  it("should transfer the canvases in the browser on Linux", async () => {
    await renderCanvas("web", "Linux");
    await waitFor(() => expect(transfer).toHaveBeenCalledTimes(3));
    expect(getContext).not.toHaveBeenCalled();
  });

  it("should transfer the canvases in the desktop app on macOS", async () => {
    await renderCanvas("tauri", "macOS");
    await waitFor(() => expect(transfer).toHaveBeenCalledTimes(3));
    expect(getContext).not.toHaveBeenCalled();
  });

  it("should transfer the canvases in the desktop app on Windows", async () => {
    await renderCanvas("tauri", "Windows");
    await waitFor(() => expect(transfer).toHaveBeenCalledTimes(3));
    expect(getContext).not.toHaveBeenCalled();
  });
});
