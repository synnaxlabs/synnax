// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";

interface DateTimeTestCase {
  name: string;
  initialValue: string;
  changeValue: string;
}

const openCalendarModal = (result: ReturnType<typeof render>): void => {
  const { container } = result;
  // The calendar icon button is the only button rendered before the modal opens
  fireEvent.click(within(container).getByRole("button"));
};

const getDayButtons = (dialog: HTMLElement): HTMLElement[] =>
  within(dialog)
    .getAllByRole("button")
    .filter((b) => /^\d+$/.test(b.textContent?.trim() ?? ""));

describe("Input.DateTime", () => {
  const testCases: DateTimeTestCase[] = [
    {
      name: "changing from 2-digit milliseconds",
      initialValue: "2025-11-03T17:44:45.500",
      changeValue: "2025-11-03T17:44:45.501",
    },
    {
      name: "changing from .809 milliseconds",
      initialValue: "2025-11-03T17:44:45.809",
      changeValue: "2025-11-03T17:44:45.808",
    },
    {
      name: "3-digit milliseconds",
      initialValue: "2025-11-03T17:44:45.808",
      changeValue: "2025-11-03T17:44:45.809",
    },
    {
      name: "changing milliseconds without blocking",
      initialValue: "2025-11-03T17:44:45.809",
      changeValue: "2025-11-03T17:44:45.810",
    },
    {
      name: "dates in summer (DST) when current date is in winter",
      initialValue: "2025-07-15T14:30:00.000",
      changeValue: "2025-07-15T14:30:00.100",
    },
    {
      name: "dates in winter when current date is in summer",
      initialValue: "2025-01-15T10:00:00.000",
      changeValue: "2025-01-15T10:00:00.100",
    },
  ];

  testCases.forEach(({ name, initialValue, changeValue }) => {
    it(`should handle ${name}`, () => {
      const handleChange = vi.fn();
      const ts = new TimeStamp(initialValue, "local");

      render(<Input.DateTime value={Number(ts.valueOf())} onChange={handleChange} />);

      const input = screen.getByRole("textbox");

      fireEvent.change(input, { target: { value: changeValue } });

      expect(handleChange).toHaveBeenCalledOnce();

      const receivedValue = handleChange.mock.calls[0][0];
      const expectedTS = new TimeStamp(changeValue, "local");
      const expectedValue = Number(expectedTS.valueOf());

      expect(receivedValue).toEqual(expectedValue);
    });
  });

  describe("leap year", () => {
    let originalTZ: string | undefined;
    beforeEach(() => {
      originalTZ = process.env.TZ;
      process.env.TZ = "UTC";
    });
    afterEach(() => {
      process.env.TZ = originalTZ;
    });

    it("should show 29 days for February in a leap year", () => {
      const handleChange = vi.fn();
      const utcNanos = Date.UTC(2024, 1, 15, 12, 0, 0, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      expect(screen.getByText("February")).toBeTruthy();
      const dialog = screen.getByRole("dialog");
      const calendarDays = getDayButtons(dialog);
      expect(calendarDays.length).toEqual(29);
      expect(calendarDays[28].textContent?.trim()).toEqual("29");
    });

    it("should show 28 days for February in a non-leap year", () => {
      const handleChange = vi.fn();
      const utcNanos = Date.UTC(2025, 1, 15, 12, 0, 0, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      expect(screen.getByText("February")).toBeTruthy();
      const dialog = screen.getByRole("dialog");
      const calendarDays = getDayButtons(dialog);
      expect(calendarDays.length).toEqual(28);
      expect(calendarDays[27].textContent?.trim()).toEqual("28");
    });
  });

  describe("timezone sensitive", () => {
    let originalTZ: string | undefined;
    beforeEach(() => {
      originalTZ = process.env.TZ;
      process.env.TZ = "Pacific/Auckland";
    });
    afterEach(() => {
      process.env.TZ = originalTZ;
    });

    it("should display the local date in the input when crossing a date boundary", () => {
      const handleChange = vi.fn();
      // 2022-12-31T23:00:00Z -> Auckland (UTC+13): 2023-01-01T12:00:00
      const utcNanos = Date.UTC(2022, 11, 31, 23, 0, 0, 0) * 1e6;
      render(<Input.DateTime value={utcNanos} onChange={handleChange} />);
      expect(screen.getByRole<HTMLInputElement>("textbox").value).toContain(
        "2023-01-01",
      );
    });

    it("should display the local month in the calendar", () => {
      const handleChange = vi.fn();
      // 2022-06-30T23:00:00Z -> Auckland (UTC+12): 2022-07-01T11:00:00
      const utcNanos = Date.UTC(2022, 5, 30, 23, 0, 0, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      expect(screen.getByText("July")).toBeTruthy();
    });

    it("should display the local year in the calendar", () => {
      const handleChange = vi.fn();
      // 2022-12-31T23:00:00Z -> Auckland (UTC+13): 2023-01-01T12:00:00
      const utcNanos = Date.UTC(2022, 11, 31, 23, 0, 0, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      expect(screen.getByText("2023")).toBeTruthy();
    });

    it("should select the local day in the calendar", () => {
      const handleChange = vi.fn();
      // 2022-06-15T23:00:00Z -> Auckland (UTC+12): 2022-06-16T11:00:00
      const utcNanos = Date.UTC(2022, 5, 15, 23, 0, 0, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      const dialog = screen.getByRole("dialog");
      const day16 = within(dialog)
        .getAllByRole("button")
        .find((b) => b.textContent?.trim() === "16");
      expect(day16).toBeTruthy();
      expect(day16?.className).toContain("outlined");
    });

    it("should select the correct local hour in the time selector", () => {
      const handleChange = vi.fn();
      // 2022-06-15T23:30:45Z -> Auckland (UTC+12): 2022-06-16T11:30:45
      const utcNanos = Date.UTC(2022, 5, 15, 23, 30, 45, 0) * 1e6;
      const result = render(
        <Input.DateTime value={utcNanos} onChange={handleChange} />,
      );
      openCalendarModal(result);
      const dialog = screen.getByRole("dialog");
      const timeLists = dialog.querySelectorAll(".pluto-time-list");
      expect(timeLists.length).toEqual(3);
      // First list is hours — local hour 11 should be selected
      const selectedHour = timeLists[0].querySelector(".pluto--selected");
      expect(selectedHour).toBeTruthy();
      expect(selectedHour?.textContent?.trim()).toEqual("11");
      // Second list is minutes — minute 30 should be selected
      const selectedMinute = timeLists[1].querySelector(".pluto--selected");
      expect(selectedMinute).toBeTruthy();
      expect(selectedMinute?.textContent?.trim()).toEqual("30");
      // Third list is seconds — second 45 should be selected
      const selectedSecond = timeLists[2].querySelector(".pluto--selected");
      expect(selectedSecond).toBeTruthy();
      expect(selectedSecond?.textContent?.trim()).toEqual("45");
    });
  });
});
