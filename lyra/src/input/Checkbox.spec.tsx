// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";

const box = (): HTMLInputElement => screen.getByRole<HTMLInputElement>("checkbox");

describe("Input.Checkbox", () => {
  it("should mark a partly made choice as indeterminate", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Input.Checkbox value={false} indeterminate onChange={onChange} />,
    );
    expect(box().indeterminate).toBe(true);
    rerender(<Input.Checkbox value onChange={onChange} />);
    expect(box().indeterminate).toBe(false);
  });

  it("should keep the mark when its owner ignores a click", () => {
    const onChange = vi.fn();
    render(<Input.Checkbox value={false} indeterminate onChange={onChange} />);
    fireEvent.click(box());
    expect(onChange).toHaveBeenCalledWith(true);
    expect(box().indeterminate).toBe(true);
  });
});
