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

import { Input } from "@/input";

const getInput = (container: HTMLElement): HTMLInputElement =>
  container.querySelector("input") as HTMLInputElement;

describe("Input.Boolean", () => {
  it("should call onChange on a primary click", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.Switch value={false} onChange={onChange} />);
    fireEvent.click(getInput(container));
    expect(onChange).toHaveBeenCalledWith(true);
  });

  it("should cancel a secondary-button click so the value cannot flip", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.Switch value={false} onChange={onChange} />);
    const input = getInput(container);
    const proceeded = fireEvent(
      input,
      new MouseEvent("auxclick", { button: 2, bubbles: true, cancelable: true }),
    );
    expect(proceeded).toBe(false);
    expect(onChange).not.toHaveBeenCalled();
  });
});
