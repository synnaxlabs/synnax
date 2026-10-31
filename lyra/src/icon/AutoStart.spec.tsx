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

import { Icon } from "@/icon";

const SLASH = "M3 3l18 18";

const paths = (container: HTMLElement): (string | null)[] =>
  Array.from(container.querySelectorAll("path"), (path) => path.getAttribute("d"));

describe("AutoStart", () => {
  it("should draw its arc and play glyph with no slash", () => {
    const { container } = render(<Icon.AutoStart />);
    expect(paths(container)).toHaveLength(2);
    expect(paths(container)).not.toContain(SLASH);
  });

  it("should draw a slash across the glyph when off", () => {
    const { container } = render(<Icon.AutoStartOff />);
    expect(paths(container)).toContain(SLASH);
    expect(container.querySelector("mask")).toBeNull();
  });
});
