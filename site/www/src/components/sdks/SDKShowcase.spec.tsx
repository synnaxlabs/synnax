// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { type Samples, SDKShowcase } from "@/components/sdks/SDKShowcase";

const sample = (lang: string): Samples["python"] => ({
  stream: `<pre>${lang}-stream</pre>`,
  write: `<pre>${lang}-write</pre>`,
  read: `<pre>${lang}-read</pre>`,
});

const SAMPLES: Samples = {
  python: sample("py"),
  typescript: sample("ts"),
  cpp: sample("cpp"),
};

const panels = (container: HTMLElement): string[] =>
  [...container.querySelectorAll(".sdks-panel")].map((panel) => {
    const label = panel.querySelector(".sdks-panel-label")!.textContent;
    return `${label}: ${panel.querySelector(".code-panel")!.textContent}`;
  });

describe("SDKShowcase", () => {
  it("should show the Python samples first", () => {
    const { container } = render(<SDKShowcase samples={SAMPLES} />);
    expect(screen.getByText("Python").className).toContain("viz-tab--active");
    expect(panels(container)).toEqual([
      "Stream: py-stream",
      "Write: py-write",
      "Read: py-read",
    ]);
  });

  it("should show every sample of the selected language", () => {
    const { container } = render(<SDKShowcase samples={SAMPLES} />);
    fireEvent.click(screen.getByText("C++"));
    expect(screen.getByText("C++").className).toContain("viz-tab--active");
    expect(screen.getByText("Python").className).not.toContain("viz-tab--active");
    expect(panels(container)).toEqual([
      "Stream: cpp-stream",
      "Write: cpp-write",
      "Read: cpp-read",
    ]);
  });
});
