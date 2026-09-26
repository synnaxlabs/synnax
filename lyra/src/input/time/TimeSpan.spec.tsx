// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { loadLanguage } from "@/input/time/suggest";
import { mockBoundingClientRect } from "@/testutil/dom";

const HOUR = Number(TimeSpan.HOUR.valueOf());

const open = (container: HTMLElement): HTMLInputElement => {
  const trigger = container.querySelector(".pluto-time-editor__trigger");
  if (trigger == null) throw new Error("no time editor trigger");
  fireEvent.click(trigger);
  return screen.getByRole<HTMLInputElement>("textbox");
};

describe("Input.TimeSpan", () => {
  beforeAll(loadLanguage);
  beforeAll(() => {
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });

  it("should not commit an unedited value finer than a microsecond", () => {
    const onChange = vi.fn();
    const { container } = render(
      <Input.TimeSpan value={2 * HOUR + 512} onChange={onChange} />,
    );
    fireEvent.keyDown(open(container), { key: "Enter" });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should stay open on Enter when the text is not a duration", () => {
    const onChange = vi.fn();
    const { container } = render(<Input.TimeSpan value={HOUR} onChange={onChange} />);
    const field = open(container);
    fireEvent.change(field, { target: { value: "5 parsecs" } });
    fireEvent.keyDown(field, { key: "Enter" });
    expect(screen.getByRole("textbox")).toBe(field);
    expect(onChange).not.toHaveBeenCalled();
  });
});
