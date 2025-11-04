// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";

describe("Input.DateTime", () => {
  it("should handle 1-digit milliseconds", () => {
    const handleChange = vi.fn();
    const initialValue = new TimeStamp([2025, 11, 3], "local")
      .add(TimeStamp.hours(17))
      .add(TimeStamp.minutes(44))
      .add(TimeStamp.seconds(45))
      .add(TimeStamp.milliseconds(500));

    const c = render(
      <Input.DateTime value={Number(initialValue.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-11-03T17:44:45.5" },
    });

    expect(handleChange).toHaveBeenCalled();
  });

  it("should handle 2-digit milliseconds", () => {
    const handleChange = vi.fn();
    const initialValue = new TimeStamp([2025, 11, 3], "local")
      .add(TimeStamp.hours(17))
      .add(TimeStamp.minutes(44))
      .add(TimeStamp.seconds(45))
      .add(TimeStamp.milliseconds(500));

    const c = render(
      <Input.DateTime value={Number(initialValue.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-11-03T17:44:45.50" },
    });

    expect(handleChange).toHaveBeenCalled();
  });

  it("should handle 3-digit milliseconds", () => {
    const handleChange = vi.fn();
    const initialValue = new TimeStamp([2025, 11, 3], "local")
      .add(TimeStamp.hours(17))
      .add(TimeStamp.minutes(44))
      .add(TimeStamp.seconds(45))
      .add(TimeStamp.milliseconds(809));

    const c = render(
      <Input.DateTime value={Number(initialValue.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-11-03T17:44:45.809" },
    });

    expect(handleChange).toHaveBeenCalled();
  });

  it("should allow changing milliseconds without blocking", () => {
    const handleChange = vi.fn();
    const initialValue = new TimeStamp([2025, 11, 3], "local")
      .add(TimeStamp.hours(17))
      .add(TimeStamp.minutes(44))
      .add(TimeStamp.seconds(45))
      .add(TimeStamp.milliseconds(809));

    const c = render(
      <Input.DateTime value={Number(initialValue.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-11-03T17:44:45.810" },
    });

    expect(handleChange).toHaveBeenCalled();
  });

  it("should handle dates in summer (DST) when current date is in winter", () => {
    const handleChange = vi.fn();
    const summerDate = new TimeStamp([2025, 7, 15], "local")
      .add(TimeStamp.hours(14))
      .add(TimeStamp.minutes(30))
      .add(TimeStamp.seconds(0))
      .add(TimeStamp.milliseconds(0));

    const c = render(
      <Input.DateTime value={Number(summerDate.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-07-15T14:30:00.000" },
    });

    expect(handleChange).toHaveBeenCalled();
  });

  it("should handle dates in winter when current date is in summer", () => {
    const handleChange = vi.fn();
    const winterDate = new TimeStamp([2025, 1, 15], "local")
      .add(TimeStamp.hours(10))
      .add(TimeStamp.minutes(0))
      .add(TimeStamp.seconds(0))
      .add(TimeStamp.milliseconds(0));

    const c = render(
      <Input.DateTime value={Number(winterDate.valueOf())} onChange={handleChange} />,
    );

    const input = c.container.querySelector(
      'input[type="datetime-local"]',
    ) as HTMLInputElement;
    expect(input).toBeTruthy();

    fireEvent.change(input, {
      target: { value: "2025-01-15T10:00:00.000" },
    });

    expect(handleChange).toHaveBeenCalled();
  });
});
