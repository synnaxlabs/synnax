// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
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
