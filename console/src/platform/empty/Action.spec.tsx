// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Empty } from "@/platform";

const keycaps = (container: HTMLElement): number =>
  container.querySelectorAll(".pluto-text--keyboard").length;

describe("Empty.Action", () => {
  it("should show the shortcut that runs the same action", () => {
    const c = render(
      <Empty.Action
        message="No components open"
        action="Create component"
        trigger={["Control", "T"]}
        onClick={vi.fn()}
      />,
    );
    // An empty state is where a new user looks first, so it is the best place to
    // teach the shortcut the action also answers to.
    expect(c.getByText("Create component")).toBeTruthy();
    expect(keycaps(c.container)).toBe(2);
  });

  it("should stay unadorned when the action has no shortcut", () => {
    const c = render(
      <Empty.Action
        message="No components open"
        action="Create component"
        onClick={vi.fn()}
      />,
    );
    expect(c.getByText("Create component")).toBeTruthy();
    expect(keycaps(c.container)).toBe(0);
  });

  it("should withhold the hint when the action itself is withheld", () => {
    // A viewer gets no action link, so a bare shortcut floating under the message
    // would advertise something they cannot do.
    const c = render(
      <Empty.Action message="No components open" trigger={["Control", "T"]} />,
    );
    expect(c.getByText("No components open")).toBeTruthy();
    expect(keycaps(c.container)).toBe(0);
  });
});
