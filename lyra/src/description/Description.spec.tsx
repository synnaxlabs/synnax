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

import { Description } from "@/description";

describe("Description", () => {
  it("should pair each label with its value in a description list", () => {
    render(
      <Description.List>
        <Description.Item>
          <Description.Label>Version</Description.Label>
          <Description.Value>0.58.0</Description.Value>
        </Description.Item>
      </Description.List>,
    );
    const label = screen.getByText("Version");
    const value = screen.getByText("0.58.0");
    expect(label.tagName).toBe("DT");
    expect(value.tagName).toBe("DD");
    expect(label.parentElement).toBe(value.parentElement);
    expect(label.closest("dl")).not.toBeNull();
  });

  it("should give labels and values the level of the list", () => {
    render(
      <Description.List level="small">
        <Description.Item>
          <Description.Label>Term</Description.Label>
          <Description.Value>Perpetual</Description.Value>
        </Description.Item>
      </Description.List>,
    );
    expect(screen.getByText("Term").classList).toContain("pluto-text--small");
    expect(screen.getByText("Perpetual").classList).toContain("pluto-text--small");
  });

  it.each(["start", "between"] as const)(
    "should mark the list with the %s justification",
    (justify) => {
      const { container } = render(
        <Description.List justify={justify}>
          <Description.Item>
            <Description.Label>Term</Description.Label>
            <Description.Value>Perpetual</Description.Value>
          </Description.Item>
        </Description.List>,
      );
      expect(container.querySelector("dl")?.classList).toContain(
        `pluto-description--${justify}`,
      );
    },
  );
});
