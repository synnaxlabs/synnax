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

import { Pane } from "@/pane";

describe("Pane", () => {
  it("should render its items inside one pane", () => {
    render(
      <Pane.Pane>
        <Pane.Item>T4</Pane.Item>
        <Pane.Item>T7-Pro</Pane.Item>
      </Pane.Pane>,
    );
    const pane = screen.getByText("T4").closest(".lyra-pane");
    expect(pane).not.toBeNull();
    expect(screen.getByText("T7-Pro").closest(".lyra-pane")).toBe(pane);
    expect(screen.getByText("T4").classList).toContain("lyra-pane__item");
  });

  it("should call onClick when a button is clicked", () => {
    const onClick = vi.fn();
    render(
      <Pane.Pane>
        <Pane.Button onClick={onClick}>LabJack hardware</Pane.Button>
      </Pane.Pane>,
    );
    fireEvent.click(screen.getByRole("button", { name: "LabJack hardware" }));
    expect(onClick).toHaveBeenCalledOnce();
  });

  it("should end a button with a caret", () => {
    render(<Pane.Button>LabJack hardware</Pane.Button>);
    const button = screen.getByRole("button", { name: "LabJack hardware" });
    expect(button.lastElementChild?.classList).toContain("lyra-pane__caret");
  });
});

describe("Pane.Disclosure", () => {
  const renderDisclosure = (open?: boolean) =>
    render(
      <Pane.Disclosure open={open}>
        <Pane.Summary>Not seeing your hardware?</Pane.Summary>
        <Pane.Item>Check the cable.</Pane.Item>
      </Pane.Disclosure>,
    );

  it("should render a native details element with its summary first", () => {
    renderDisclosure();
    const summary = screen.getByText("Not seeing your hardware?").closest("summary");
    expect(summary).not.toBeNull();
    const details = summary?.parentElement;
    expect(details?.tagName).toBe("DETAILS");
    expect(details?.firstElementChild).toBe(summary);
  });

  it("should start closed", () => {
    renderDisclosure();
    const details = screen.getByText("Check the cable.").closest("details");
    expect(details?.open).toBe(false);
  });

  it("should start open when asked", () => {
    renderDisclosure(true);
    const details = screen.getByText("Check the cable.").closest("details");
    expect(details?.open).toBe(true);
  });

  it("should open when the summary is clicked", () => {
    renderDisclosure();
    fireEvent.click(screen.getByText("Not seeing your hardware?"));
    const details = screen.getByText("Check the cable.").closest("details");
    expect(details?.open).toBe(true);
  });
});
