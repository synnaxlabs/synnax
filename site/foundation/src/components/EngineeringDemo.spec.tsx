// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { EngineeringDemo } from "@/components/EngineeringDemo";

describe("EngineeringDemo", () => {
  afterEach(cleanup);

  it("should explain bounded buffering during an interruption and distinguish telemetry recovery from expired commands", async () => {
    const { container } = render(<EngineeringDemo />);
    await screen.findByRole("img");
    const explanation = within(container.querySelector('[aria-live="polite"]')!);

    fireEvent.click(screen.getByRole("button", { name: "Interrupt the link" }));
    expect(
      screen.getByRole("img", {
        name: /Distributed reliability, step 2.*uplink is interrupted.*collect data into its buffer/i,
      }),
    ).toBeTruthy();
    expect(explanation.getByText(/Retention and disk capacity define/)).toBeTruthy();
    expect(explanation.getByText(/Command deadlines still apply/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Restore the link" }));
    expect(
      screen.getByRole("img", {
        name: /Distributed reliability, step 3.*connection is restored.*Buffered data moves/i,
      }),
    ).toBeTruthy();
    expect(explanation.getByText(/Expired commands stay expired/)).toBeTruthy();
    expect(explanation.queryByText(/Retention and disk capacity define/)).toBeNull();
    expect(
      screen.getByRole("button", { name: "Restore the link", pressed: true }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Interrupt the link", pressed: false }),
    ).toBeTruthy();
  });

  it("should reset the scenario when changing attributes so a faulted state does not carry into another diagram", async () => {
    render(<EngineeringDemo />);
    await screen.findByRole("img");
    const attributes = within(
      screen.getByRole("group", { name: "Engineering attributes" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Interrupt the link" }));

    fireEvent.click(attributes.getByRole("button", { name: "Performance" }));
    expect(attributes.getAllByRole("button", { pressed: true })).toEqual([
      attributes.getByRole("button", { name: "Performance" }),
    ]);
    const performance = within(
      screen.getByRole("group", { name: "performance scenarios" }),
    );
    expect(performance.getAllByRole("button", { pressed: true })).toEqual([
      performance.getByRole("button", { name: "Ingest" }),
    ]);
    expect(screen.getByRole("img", { name: /^The data path, step 1/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Interrupt the link" })).toBeNull();

    fireEvent.click(performance.getByRole("button", { name: "Deliver" }));
    fireEvent.click(attributes.getByRole("button", { name: "Deployment" }));
    expect(attributes.getAllByRole("button", { pressed: true })).toEqual([
      attributes.getByRole("button", { name: "Deployment" }),
    ]);
    const deployment = within(
      screen.getByRole("group", { name: "deployment scenarios" }),
    );
    expect(deployment.getAllByRole("button", { pressed: true })).toEqual([
      deployment.getByRole("button", { name: "One node" }),
    ]);
    expect(
      screen.getByRole("img", { name: /^Deployment topology, step 1/ }),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Deliver" })).toBeNull();
  });
});
