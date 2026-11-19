// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("macOS");
});

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "tauri" }));
vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { type Haul } from "@synnaxlabs/lyra/haul";
import { clearMocks, mockIPC } from "@tauri-apps/api/mocks";
import { act, fireEvent, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { ITEM, renderDropOutside, SOURCE } from "@/platform/window/testutil";

const RELEASE: [number, number] = [100, 360];
const TARGET: Haul.Item = { key: "mosaic", type: "mosaic" };

/** Answers the Console's mouse_release calls with a release it holds until told. */
const interceptRelease = () => {
  const release = Promise.withResolvers<[number, number]>();
  const mouseRelease = vi.fn(async () => await release.promise);
  mockIPC((cmd) => (cmd === "mouse_release" ? mouseRelease() : undefined));
  onTestFinished(clearMocks);
  return { mouseRelease, release: release.resolve };
};

describe("useDropOutside on macOS", () => {
  it("should drop a drag released over the desktop at the native cursor", async () => {
    const { release } = interceptRelease();
    const { onDrop, haul } = await renderDropOutside();
    act(() => haul?.start(SOURCE, [ITEM]));
    fireEvent.dragStart(document.body);
    await act(async () => release(RELEASE));
    await waitFor(() =>
      expect(onDrop).toHaveBeenCalledWith(expect.anything(), { x: 100, y: 360 }),
    );
  });

  it("should not tear off a drag that already dropped inside the window", async () => {
    const { release } = interceptRelease();
    const { onDrop, haul } = await renderDropOutside();
    act(() => haul?.start(SOURCE, [ITEM]));
    fireEvent.dragStart(document.body);
    act(() => haul?.drop({ target: TARGET, dropped: [ITEM] }));
    await act(async () => release(RELEASE));
    expect(onDrop).not.toHaveBeenCalled();
  });

  it("should not read the cursor until a drag starts", async () => {
    const { mouseRelease } = interceptRelease();
    await renderDropOutside();
    await act(async () => {});
    expect(mouseRelease).not.toHaveBeenCalled();
    fireEvent.dragStart(document.body);
    expect(mouseRelease).toHaveBeenCalledOnce();
  });
});
