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

const dashed = (): boolean =>
  box().closest("label")?.classList.contains("pluto--indeterminate") ?? false;

describe("Input.Checkbox", () => {
  it("should mark a partly made choice for screen readers and draw a dash", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Input.Checkbox value={false} indeterminate onChange={onChange} />,
    );
    expect(box().indeterminate).toBe(true);
    expect(dashed()).toBe(true);
    rerender(<Input.Checkbox value onChange={onChange} />);
    expect(box().indeterminate).toBe(false);
    expect(dashed()).toBe(false);
  });

  it("should set the flag again on the render after a click clears it", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Input.Checkbox value={false} indeterminate onChange={onChange} />,
    );
    fireEvent.click(box());
    expect(onChange).toHaveBeenCalledWith(true);
    expect(dashed()).toBe(true);
    rerender(<Input.Checkbox value={false} indeterminate onChange={onChange} />);
    expect(box().indeterminate).toBe(true);
  });
});
