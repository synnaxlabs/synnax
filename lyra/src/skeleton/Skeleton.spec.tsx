// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Skeleton } from "@/skeleton/Skeleton";

describe("Skeleton", () => {
  it("should hold the size of the content it stands in for", () => {
    const { container } = render(<Skeleton width="10rem" height="2rem" />);
    const block = container.firstElementChild;
    if (!(block instanceof HTMLElement)) throw new Error("no skeleton block");
    expect(block.style.width).toBe("10rem");
    expect(block.style.height).toBe("2rem");
  });
});
