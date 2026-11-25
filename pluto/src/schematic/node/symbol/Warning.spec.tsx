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

import { Warning } from "@/schematic/node/symbol/Warning";

describe("Warning", () => {
  it.each([
    ["missing", "Missing symbol"],
    ["failed", "Failed to load symbol"],
    ["invalid", "Invalid symbol"],
  ] as const)("should show the text for a %s symbol", (cause, text) => {
    const { getByText } = render(<Warning cause={cause} />);
    expect(getByText(text)).toBeTruthy();
  });

  it("should set its class name on the note", () => {
    const { container } = render(<Warning cause="missing" className="node" />);
    expect(container.firstElementChild?.classList.contains("node")).toBe(true);
  });
});
