// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

export const mockBoundingClientRect = (
  top: number,
  left: number,
  width: number,
  height: number,
): Mock<typeof HTMLElement.prototype.getBoundingClientRect> =>
  vi.fn().mockReturnValue({
    top,
    left,
    width,
    height,
    bottom: top + height,
    right: left + width,
    x: 0,
    y: 0,
    toJSON: () => "",
  });
