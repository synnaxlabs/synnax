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

interface DateTimeTestCase {
  name: string;
  initialValue: string;
  changeValue: string;
}

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

      const c = render(
        <Input.DateTime value={Number(ts.valueOf())} onChange={handleChange} />,
      );

      const input = c.container.querySelector(
        'input[type="datetime-local"]',
      ) as HTMLInputElement;
      expect(input).toBeTruthy();

      fireEvent.change(input, { target: { value: changeValue } });

      expect(handleChange).toHaveBeenCalledOnce();

      const receivedValue = handleChange.mock.calls[0][0];
      const expectedTS = new TimeStamp(changeValue, "local");
      const expectedValue = Number(expectedTS.valueOf());

      expect(receivedValue).toEqual(expectedValue);
    });
  });
});
