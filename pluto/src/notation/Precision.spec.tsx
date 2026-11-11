// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Notation } from "@/notation";

const commit = (value: string): ReturnType<typeof vi.fn> => {
  const onChange = vi.fn();
  const { getByRole } = render(<Notation.Precision value={2} onChange={onChange} />);
  const input = getByRole("textbox");
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
  return onChange;
};

describe("Notation.Precision", () => {
  it("should round a typed fraction to a whole number", () => {
    expect(commit("2.6")).toHaveBeenLastCalledWith(3);
  });

  it("should clamp a value above 17", () => {
    expect(commit("40")).toHaveBeenLastCalledWith(17);
  });
});
