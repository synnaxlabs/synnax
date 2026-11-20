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
  pinOS("Linux");
});

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));
vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import type { xy } from "@synnaxlabs/x";
import { act } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ITEM, renderDropOutside, SOURCE } from "@/platform/window/testutil";

const CURSOR: xy.XY = { x: 100, y: 360 };

describe("useDropOutside", () => {
  it("should resolve a drag ending over the desktop under Tauri", async () => {
    mocks.engine = "tauri";
    const { onDrop, haul } = await renderDropOutside();
    act(() => {
      haul?.start(SOURCE, [ITEM]);
      haul?.end(CURSOR);
    });
    expect(onDrop).toHaveBeenCalledOnce();
  });

  // The web runtime reports no window geometry, so an unclaimed drop anywhere in the
  // page looks like one released over the desktop and tears the tab out of view.
  it("should leave an unclaimed drop alone in the browser", async () => {
    mocks.engine = "web";
    const { onDrop, haul } = await renderDropOutside();
    act(() => {
      haul?.start(SOURCE, [ITEM]);
      haul?.end(CURSOR);
    });
    expect(onDrop).not.toHaveBeenCalled();
  });
});
