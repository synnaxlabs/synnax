// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ResizeObserver } from "@juggle/resize-observer";
import { afterAll, beforeAll, vi } from "vitest";

import { installTestWebSocket } from "@/testutil/websocket";

class MockIntersectionObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
}

// Installed at module scope: an async describe body can open a socket at
// collection time, before any beforeAll runs.
installTestWebSocket();

beforeAll(() => {
  vi.stubGlobal("ResizeObserver", ResizeObserver);
  vi.stubGlobal("OffscreenCanvas", {});
  vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
  // jsdom does not implement the pointer capture APIs that Cursor.useDrag relies on.
  HTMLElement.prototype.setPointerCapture = () => {};
  HTMLElement.prototype.releasePointerCapture = () => {};
  HTMLElement.prototype.hasPointerCapture = () => false;
  // jsdom does not implement scrollIntoView; Tabs.Selector calls it to reveal the
  // selected tab.
  Element.prototype.scrollIntoView = () => {};
});

afterAll(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});
