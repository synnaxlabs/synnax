// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { Input } from "@/input";

describe("Input.Checkbox real input check", () => {
  it("should keep a disabled check on a real pointer click", async () => {
    const onChange = vi.fn();
    render(<Input.Checkbox value onChange={onChange} disabled />);
    const input = screen.getByRole("checkbox");
    await userEvent.click(input, { force: true });
    const label = input.closest("label");
    expect(label).not.toBeNull();
    if (label != null) await userEvent.click(label, { force: true });
    expect(onChange).not.toHaveBeenCalled();
    expect(input).toHaveProperty("checked", true);
  });

  it("should keep a disabled check on Space", async () => {
    const onChange = vi.fn();
    render(<Input.Checkbox value onChange={onChange} disabled />);
    const input = screen.getByRole("checkbox");
    input.focus();
    await userEvent.keyboard(" ");
    expect(onChange).not.toHaveBeenCalled();
    expect(input).toHaveProperty("checked", true);
  });
});
