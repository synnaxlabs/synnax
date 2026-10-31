// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mockBoundingClientRect } from "@synnaxlabs/lyra/testutil";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Problem } from "@/vis/problem";
import { type problem } from "@/vis/problem/aether";

const DELAY = TimeSpan.milliseconds(100);

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config delay={DELAY}>{children}</Tooltip.Config>
  </Triggers.Provider>
);

const FAILED: problem.Problem = {
  variant: "error",
  name: "PT-03",
  message: "Failed to read channel data",
  description: "channel 999 not found",
};

const RECONNECTING: problem.Problem = {
  variant: "warning",
  name: "PT-04",
  message: "Stream reconnecting",
  description: "",
};

describe("Problem.Mark", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("should render nothing when there are no problems", () => {
    const { container } = render(<Problem.Mark problems={[]} />, { wrapper: Wrapper });
    expect(container.innerHTML).toBe("");
  });

  it("should show the glyph of the worst problem", () => {
    render(<Problem.Mark problems={[RECONNECTING, FAILED]} />, { wrapper: Wrapper });
    const mark = screen.getByRole("img", { name: "Failed to read channel data" });
    expect(mark.querySelector(".lyra-icon--error")).not.toBeNull();
    expect(mark.querySelector(".lyra-icon--warning")).toBeNull();
  });

  it("should show the warning glyph when every problem is a warning", () => {
    render(<Problem.Mark problems={[RECONNECTING]} />, { wrapper: Wrapper });
    const mark = screen.getByRole("img", { name: "Stream reconnecting" });
    expect(mark.querySelector(".lyra-icon--warning")).not.toBeNull();
  });

  it("should frame the glyph in a tile by default", () => {
    render(<Problem.Mark problems={[FAILED]} />, { wrapper: Wrapper });
    const mark = screen.getByRole("img");
    expect(mark.classList).toContain("pluto-problem-mark--tile");
    expect(mark.querySelector(".lyra-icon-tile > .lyra-icon--error")).not.toBeNull();
  });

  it("should show the glyph without a tile for the glyph and badge variants", () => {
    const { rerender } = render(<Problem.Mark problems={[FAILED]} variant="glyph" />, {
      wrapper: Wrapper,
    });
    expect(screen.getByRole("img").classList).toContain("pluto-problem-mark--glyph");
    expect(screen.getByRole("img").querySelector(".lyra-icon-tile")).toBeNull();
    rerender(<Problem.Mark problems={[FAILED]} variant="badge" />);
    expect(screen.getByRole("img").classList).toContain("pluto-problem-mark--badge");
    expect(screen.getByRole("img").querySelector(".lyra-icon-tile")).toBeNull();
  });

  it("should list the name, message, and description of each problem on hover", () => {
    render(<Problem.Mark problems={[FAILED, RECONNECTING]} />, { wrapper: Wrapper });
    const mark = screen.getByRole("img");
    mark.getBoundingClientRect = mockBoundingClientRect(100, 100, 18, 18);
    fireEvent.pointerOver(mark, { pointerType: "mouse" });
    act(() => {
      vi.advanceTimersByTime(DELAY.milliseconds + 10);
    });
    expect(screen.getByRole("tooltip").textContent).toBe(
      "PT-03Failed to read channel datachannel 999 not foundPT-04Stream reconnecting",
    );
  });
});
