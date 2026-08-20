// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ResizeObserver } from "@juggle/resize-observer";
import { configure } from "@testing-library/react";
import { afterAll, beforeAll, vi } from "vitest";

import { installTestWebSocket } from "@/testutil/websocket";

// Live-core round-trips share the single test cluster with the rest of the suite, so
// allow more than the 1s waitFor default.
configure({ asyncUtilTimeout: 5000 });

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
  HTMLElement.prototype.setPointerCapture = () => {};
  HTMLElement.prototype.releasePointerCapture = () => {};
  HTMLElement.prototype.hasPointerCapture = () => false;
  Element.prototype.scrollIntoView = () => {};
});

afterAll(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});
