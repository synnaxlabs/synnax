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

import { Input } from "@/input";

const dashed = (): boolean =>
  screen
    .getByRole("checkbox")
    .closest("label")
    ?.classList.contains("lyra--indeterminate") ?? false;

describe("Input.Checkbox", () => {
  it("should draw a dash for a partly made choice", () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <Input.Checkbox value={false} indeterminate onChange={onChange} />,
    );
    expect(dashed()).toBe(true);
    rerender(<Input.Checkbox value onChange={onChange} />);
    expect(dashed()).toBe(false);
  });
});
