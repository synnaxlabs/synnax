// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Steps } from "@/steps";

describe("Steps", () => {
  it("should render the items in order inside an ordered list", () => {
    render(
      <Steps.List>
        <Steps.Item>Install the Driver.</Steps.Item>
        <Steps.Item>Plug in the device.</Steps.Item>
      </Steps.List>,
    );
    const list = screen.getByRole("list");
    expect(list.tagName).toBe("OL");
    expect(screen.getAllByRole("listitem").map((i) => i.textContent)).toEqual([
      "Install the Driver.",
      "Plug in the device.",
    ]);
  });

  it.each(["p", "small"] as const)(
    "should mark the list with the %s level",
    (level) => {
      render(
        <Steps.List level={level}>
          <Steps.Item>Install the Driver.</Steps.Item>
        </Steps.List>,
      );
      expect(screen.getByRole("list").classList).toContain(`lyra-steps--${level}`);
    },
  );

  it("should default to the p level", () => {
    render(
      <Steps.List>
        <Steps.Item>Install the Driver.</Steps.Item>
      </Steps.List>,
    );
    expect(screen.getByRole("list").classList).toContain("lyra-steps--p");
  });

  it("should keep the start of a list that continues after a block", () => {
    render(
      <Steps.List start={3}>
        <Steps.Item>Open the task.</Steps.Item>
      </Steps.List>,
    );
    expect(screen.getByRole("list").getAttribute("start")).toBe("3");
  });
});
