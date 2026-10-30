// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Ranger } from "@/ranger";

const range = (start: TimeSpan, end: TimeSpan): TimeRange => {
  const now = TimeStamp.now();
  return new TimeRange(now.add(start), now.add(end));
};

describe("Ranger.StageIcon", () => {
  it("should label the icon with the stage of the range", () => {
    const c = render(
      <>
        <Ranger.StageIcon timeRange={range(TimeSpan.hours(-2), TimeSpan.hours(-1))} />
        <Ranger.StageIcon timeRange={range(TimeSpan.hours(-1), TimeSpan.hours(1))} />
        <Ranger.StageIcon timeRange={range(TimeSpan.hours(1), TimeSpan.hours(2))} />
      </>,
    );
    expect(c.getByLabelText("Completed")).toBeDefined();
    expect(c.getByLabelText("In progress")).toBeDefined();
    expect(c.getByLabelText("To do")).toBeDefined();
  });

  it("should take a label from its caller", () => {
    const c = render(
      <Ranger.StageIcon
        timeRange={range(TimeSpan.hours(-2), TimeSpan.hours(-1))}
        aria-label="Stage"
      />,
    );
    expect(c.getByLabelText("Stage")).toBeDefined();
  });
});
