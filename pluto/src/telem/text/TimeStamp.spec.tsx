// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp as XTimeStamp } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("TimeStamp", () => {
  it("should render timestamp with default format", () => {
    const ts = new XTimeStamp([2024, 1, 1], "UTC")
      .add(TimeSpan.hours(12))
      .add(TimeSpan.minutes(30))
      .add(TimeSpan.seconds(45));
    const c = render(
      <Telem.Text.TimeStamp suppliedTZ="UTC" displayTZ="UTC">
        {ts}
      </Telem.Text.TimeStamp>,
    );
    expect(c.getByText("Jan 1 12:30:45")).toBeTruthy();
  });
  it("should render timestamp with time format", () => {
    const ts = new XTimeStamp(
      TimeSpan.hours(12).add(TimeSpan.minutes(30)).add(TimeSpan.seconds(45)),
      "UTC",
    );
    const c = render(
      <Telem.Text.TimeStamp format="time" suppliedTZ="UTC" displayTZ="UTC">
        {ts}
      </Telem.Text.TimeStamp>,
    );
    expect(c.getByText("12:30:45")).toBeTruthy();
  });
  it("should handle timezone conversion", () => {
    const ts = XTimeStamp.ZERO;
    const c = render(
      <Telem.Text.TimeStamp suppliedTZ="UTC" displayTZ="UTC" format="time">
        {ts}
      </Telem.Text.TimeStamp>,
    );
    expect(c.getByText("00:00:00")).toBeTruthy();
  });
  it("should accept number as microseconds", () => {
    const ts = TimeSpan.hours(12)
      .add(TimeSpan.minutes(30))
      .add(TimeSpan.seconds(45))
      .valueOf();
    const c = render(
      <Telem.Text.TimeStamp suppliedTZ="UTC" displayTZ="UTC" format="time">
        {ts}
      </Telem.Text.TimeStamp>,
    );
    expect(c.getByText("12:30:45")).toBeTruthy();
  });
  it("should pass through text props", () => {
    const ts = new XTimeStamp([2024, 1, 1], "UTC");
    const c = render(
      <Telem.Text.TimeStamp level="h1" color={5} suppliedTZ="UTC" displayTZ="UTC">
        {ts}
      </Telem.Text.TimeStamp>,
    );
    expect(c.container.querySelector("h1")).toBeTruthy();
  });
});
