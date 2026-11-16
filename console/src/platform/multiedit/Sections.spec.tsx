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
import { assert, describe, expect, it, vi } from "vitest";

import { MultiEdit } from "@/platform/multiedit";

describe("NumberFormatSection", () => {
  it("should round a typed fraction to a whole precision", () => {
    const onPrecisionChange = vi.fn();
    const { getByText } = render(
      <MultiEdit.NumberFormatSection
        onNotationChange={vi.fn()}
        onPrecisionChange={onPrecisionChange}
      />,
    );
    const input = getByText("Precision")
      .closest(".lyra-input__item")
      ?.querySelector("input");
    assert(input != null);
    fireEvent.change(input, { target: { value: "2.6" } });
    fireEvent.blur(input);
    expect(onPrecisionChange).toHaveBeenLastCalledWith(3);
  });
});

describe("StalenessSection", () => {
  const COLORS = [{ fallback: "#000000" }];

  const field = (): HTMLInputElement => {
    const input = screen
      .getByText("Timeout")
      .closest(".lyra-input__item")
      ?.querySelector("input");
    assert(input != null);
    return input;
  };

  const typeTimeout = (value: string): void => {
    fireEvent.focus(field());
    fireEvent.change(field(), { target: { value } });
    fireEvent.blur(field());
  };

  it("should show and write a timeout in nanoseconds by default", () => {
    const onTimeoutChange = vi.fn();
    render(
      <MultiEdit.StalenessSection
        colors={COLORS}
        timeout={Number(TimeSpan.seconds(3).valueOf())}
        onColorChange={vi.fn()}
        onTimeoutChange={onTimeoutChange}
      />,
    );
    expect(field().value).toBe("3s");
    typeTimeout("7s");
    expect(onTimeoutChange).toHaveBeenLastCalledWith(
      Number(TimeSpan.seconds(7).valueOf()),
    );
  });

  it("should show and write a timeout in the given unit", () => {
    const onTimeoutChange = vi.fn();
    render(
      <MultiEdit.StalenessSection
        colors={COLORS}
        timeout={3}
        unit={TimeSpan.SECOND}
        onColorChange={vi.fn()}
        onTimeoutChange={onTimeoutChange}
      />,
    );
    expect(field().value).toBe("3s");
    typeTimeout("7s");
    expect(onTimeoutChange).toHaveBeenLastCalledWith(7);
  });

  it("should show the default timeout when no element has one", () => {
    render(
      <MultiEdit.StalenessSection
        colors={COLORS}
        unit={TimeSpan.SECOND}
        onColorChange={vi.fn()}
        onTimeoutChange={vi.fn()}
      />,
    );
    expect(field().value).toBe("5s");
  });
});
