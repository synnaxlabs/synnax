// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterAll, beforeAll, vi } from "vitest";

beforeAll(() => {
  vi.stubGlobal(
    "ResizeObserver",
    vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      unobserve: vi.fn(),
      disconnect: vi.fn(),
    })),
  );
  vi.stubGlobal(
    "IntersectionObserver",
    vi.fn().mockImplementation(() => ({
      observe: vi.fn(),
      unobserve: vi.fn(),
      disconnect: vi.fn(),
    })),
  );
  Element.prototype.getBoundingClientRect = vi.fn().mockReturnValue({
    top: 0,
    left: 0,
    width: 100,
    height: 100,
    bottom: 100,
    right: 100,
    x: 0,
    y: 0,
    toJSON: () => "",
  });
});

afterAll(() => {
  vi.clearAllMocks();
});
