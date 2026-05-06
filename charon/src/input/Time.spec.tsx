// Copyright 2026 Synnax Labs, Inc.
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

describe("Input.Time", () => {
  it("should allow the user to input a time", () => {
    const handleChange = vi.fn();
    const c = render(<Input.Time value={0} onChange={handleChange} tzInfo="UTC" />);
    fireEvent.change(c.getByDisplayValue("00:00:00"), {
      target: { value: "15:00:00" },
    });
    expect(handleChange).toHaveBeenCalledWith(
      Number(TimeStamp.hours(15).add(TimeStamp.utcOffset).valueOf()),
    );
  });
});
