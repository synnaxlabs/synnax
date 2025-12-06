// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ResizeObserver } from "@juggle/resize-observer";
import { afterAll, beforeAll, vi } from "vitest";

const MockIntersectionObserver = vi.fn(() => ({
  observe: vi.fn(),
  disconnect: vi.fn(),
  unobserve: vi.fn(),
}));

beforeAll(() => {
  vi.stubGlobal("ResizeObserver", ResizeObserver);
  vi.stubGlobal("OffscreenCanvas", {} as OffscreenCanvas);
  vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
});

afterAll(() => {
  vi.clearAllMocks();
});
