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

import { Disclosure } from "@/disclosure";

const renderDisclosure = (open?: boolean) =>
  render(
    <Disclosure.Disclosure open={open}>
      <Disclosure.Summary>Not seeing your hardware?</Disclosure.Summary>
      <Disclosure.Content>Install the Driver.</Disclosure.Content>
    </Disclosure.Disclosure>,
  );

describe("Disclosure", () => {
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
    const details = screen.getByText("Install the Driver.").closest("details");
    expect(details?.open).toBe(false);
  });

  it("should start open when asked", () => {
    renderDisclosure(true);
    const details = screen.getByText("Install the Driver.").closest("details");
    expect(details?.open).toBe(true);
  });

  it("should open when the summary is clicked", () => {
    renderDisclosure();
    fireEvent.click(screen.getByText("Not seeing your hardware?"));
    const details = screen.getByText("Install the Driver.").closest("details");
    expect(details?.open).toBe(true);
  });
});
