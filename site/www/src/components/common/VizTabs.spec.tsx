// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { VizTabs } from "@/components/common/VizTabs";

const Glyph = () => <svg data-testid="glyph" />;

const TABS = [
  { key: "a", title: "Alpha", icon: Glyph },
  { key: "b", title: "Beta" },
];

describe("VizTabs", () => {
  it("should mark only the active tab", () => {
    render(<VizTabs tabs={TABS} active={1} onSelect={vi.fn()} />);
    expect(screen.getByText("Alpha").className).toBe("viz-tab");
    expect(screen.getByText("Beta").className).toBe("viz-tab viz-tab--active");
  });

  it("should report the index of the clicked tab", () => {
    const onSelect = vi.fn();
    render(<VizTabs tabs={TABS} active={0} onSelect={onSelect} />);
    fireEvent.click(screen.getByText("Beta"));
    expect(onSelect).toHaveBeenCalledExactlyOnceWith(1);
  });

  it("should show the icon of a tab that has one", () => {
    render(<VizTabs tabs={TABS} active={0} onSelect={vi.fn()} />);
    expect(screen.getAllByTestId("glyph")).toHaveLength(1);
    expect(screen.getByText("Alpha").contains(screen.getByTestId("glyph"))).toBe(true);
  });
});
