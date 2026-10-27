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

import { Text } from "@/components/text";

const LIMIT = 32;

const code = (text: string): HTMLElement => {
  const { container } = render(<Text.InlineCode text={text} />);
  return container.querySelector("code") as HTMLElement;
};

describe("InlineCode", () => {
  it("should mark a short token", () => {
    expect(code("Use `->` for values").hasAttribute("data-short")).toBe(true);
  });

  it("should leave code with spaces free to wrap", () => {
    expect(code("`must be exactly 32 characters`").hasAttribute("data-short")).toBe(
      false,
    );
  });

  it("should leave a token over the length limit free to wrap", () => {
    const token = "a".repeat(LIMIT + 1);
    expect(code(`\`${token}\``).hasAttribute("data-short")).toBe(false);
  });
});
