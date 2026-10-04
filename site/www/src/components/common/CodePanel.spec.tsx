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

import { CodePanel } from "@/components/common/CodePanel";

const HTML =
  '<pre><code><span class="line">a</span>\n<span class="line">b</span>\n' +
  '<span class="line">c</span></code></pre>';

const active = (container: HTMLElement): string[] =>
  [...container.querySelectorAll('.line[data-active="true"]')].map(
    (el) => el.textContent ?? "",
  );

describe("CodePanel", () => {
  it("should highlight the given lines, counting from one", () => {
    const { container } = render(<CodePanel html={HTML} activeLines={[1, 3]} />);
    expect(active(container)).toEqual(["a", "c"]);
    expect(container.querySelectorAll(".line")).toHaveLength(3);
  });

  it("should highlight no line when none is active", () => {
    const { container } = render(<CodePanel html={HTML} activeLines={[]} />);
    expect(active(container)).toEqual([]);
  });

  it("should ignore a line past the end of the code", () => {
    const { container } = render(<CodePanel html={HTML} activeLines={[4]} />);
    expect(active(container)).toEqual([]);
  });

  it("should move the highlight when the active lines change", () => {
    const { container, rerender } = render(<CodePanel html={HTML} activeLines={[1]} />);
    rerender(<CodePanel html={HTML} activeLines={[2]} />);
    expect(active(container)).toEqual(["b"]);
  });

  it("should add the class next to its own", () => {
    const { container } = render(
      <CodePanel html={HTML} activeLines={[]} className="viz-code" />,
    );
    expect(container.firstElementChild!.className).toBe("code-panel viz-code");
  });
});
