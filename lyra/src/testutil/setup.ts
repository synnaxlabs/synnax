// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "zod/compile";

import { ResizeObserver } from "@juggle/resize-observer";
import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";
import { configure } from "@testing-library/react";
import { afterAll, beforeAll, vi } from "vitest";

configure({ asyncUtilTimeout: 5000 });

failOnConsoleOutput();

class MockIntersectionObserver {
  observe = vi.fn();
  disconnect = vi.fn();
  unobserve = vi.fn();
}

beforeAll(() => {
  // A polyfill, not a stub, so a spec that calls vi.unstubAllGlobals keeps it.
  Object.assign(globalThis, { ResizeObserver });
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
