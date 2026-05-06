// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan as XTimeSpan } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Telem } from "@/telem";

describe("TimeSpan", () => {
  it("should render timespan with default format", () => {
    const span = XTimeSpan.seconds(90);
    const c = render(<Telem.Text.TimeSpan>{span}</Telem.Text.TimeSpan>);
    expect(c.getByText("1m 30s")).toBeTruthy();
  });
  it("should render timespan with hours", () => {
    const span = XTimeSpan.hours(2.5);
    const c = render(<Telem.Text.TimeSpan>{span}</Telem.Text.TimeSpan>);
    expect(c.getByText("2h 30m")).toBeTruthy();
  });
  it("should render timespan with days", () => {
    const span = XTimeSpan.days(1.5);
    const c = render(<Telem.Text.TimeSpan>{span}</Telem.Text.TimeSpan>);
    expect(c.getByText("1d 12h")).toBeTruthy();
  });
  it("should render zero timespan", () => {
    const span = XTimeSpan.ZERO;
    const c = render(<Telem.Text.TimeSpan>{span}</Telem.Text.TimeSpan>);
    expect(c.container.textContent).toBe("");
  });
  it("should accept number timespan in microseconds", () => {
    const span = XTimeSpan.seconds(90).valueOf();
    const c = render(<Telem.Text.TimeSpan>{span}</Telem.Text.TimeSpan>);
    expect(c.getByText("1m 30s")).toBeTruthy();
  });
  it("should pass through text props", () => {
    const span = XTimeSpan.seconds(60);
    const c = render(
      <Telem.Text.TimeSpan level="h2" color={3}>
        {span}
      </Telem.Text.TimeSpan>,
    );
    expect(c.container.querySelector("h2")).toBeTruthy();
  });
});
